package gtd

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kabanos/backend/internal/auth"
	"github.com/kabanos/backend/internal/httpx"
	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/validate"
)

// ---------- domain ----------

// GraphNode is one node on a board (a project's canvas, or the root board).
// A node references a task (item_id), a sub-project (ref_project_id) or is a
// free-form note. Label is the resolved display text.
type GraphNode struct {
	ID           uuid.UUID
	Kind         string // task | project | note
	ItemID       *uuid.UUID
	RefProjectID *uuid.UUID
	Label        string
	Done         bool    // task completion, for display
	Status       string  // sub-project status, for display
	Deadline     *string // YYYY-MM-DD
	Color        string  // computed: green | yellow | red
	Note         string
	ImageKeys    []string
	ImageURLs    []string // computed (presigned)
	X            float64
	Y            float64
}

// Settings holds the per-user deadline colour thresholds.
type Settings struct {
	SoonDays  int
	GraceDays int
}

func defaultSettings() Settings { return Settings{SoonDays: 3, GraceDays: 1} }

// Summary is the colour breakdown of a board and its projected completion (the
// latest deadline anywhere in the board, recursively).
type Summary struct {
	Red       int
	Yellow    int
	Green     int
	Color     string
	Projected *string
}

// BoardInfo is one graph in the colour-highlighted list of graphs.
type BoardInfo struct {
	ProjectID *uuid.UUID
	Title     string
	Summary   Summary
}

type GraphEdge struct {
	ID     uuid.UUID
	Source uuid.UUID
	Target uuid.UUID
}

// ---------- repo ----------

const graphNodeSelect = `
	SELECT n.id, n.kind, n.item_id, n.ref_project_id,
		COALESCE(NULLIF(n.label,''), i.title, p.title, ''),
		COALESCE(i.done, false), COALESCE(p.status, ''), to_char(n.deadline,'YYYY-MM-DD'), n.note, n.image_keys, n.x, n.y
	FROM gtd_graph_nodes n
	LEFT JOIN gtd_items i ON i.id = n.item_id
	LEFT JOIN gtd_projects p ON p.id = n.ref_project_id`

// ListGraph returns the nodes and edges of a board (board = nil is the root).
func (r *Repo) ListGraph(ctx context.Context, userID uuid.UUID, board *uuid.UUID) ([]GraphNode, []GraphEdge, error) {
	nrows, err := r.db.Read().Query(ctx, graphNodeSelect+
		` WHERE n.user_id=$1 AND n.board_project_id IS NOT DISTINCT FROM $2 ORDER BY n.created_at`, userID, board)
	if err != nil {
		return nil, nil, err
	}
	nodes := []GraphNode{}
	for nrows.Next() {
		var n GraphNode
		if err := nrows.Scan(&n.ID, &n.Kind, &n.ItemID, &n.RefProjectID, &n.Label, &n.Done, &n.Status, &n.Deadline, &n.Note, &n.ImageKeys, &n.X, &n.Y); err != nil {
			nrows.Close()
			return nil, nil, err
		}
		nodes = append(nodes, n)
	}
	nrows.Close()
	if err := nrows.Err(); err != nil {
		return nil, nil, err
	}

	erows, err := r.db.Read().Query(ctx,
		`SELECT id, source_id, target_id FROM gtd_graph_edges
		 WHERE user_id=$1 AND board_project_id IS NOT DISTINCT FROM $2 ORDER BY created_at`, userID, board)
	if err != nil {
		return nil, nil, err
	}
	defer erows.Close()
	edges := []GraphEdge{}
	for erows.Next() {
		var e GraphEdge
		if err := erows.Scan(&e.ID, &e.Source, &e.Target); err != nil {
			return nil, nil, err
		}
		edges = append(edges, e)
	}
	return nodes, edges, erows.Err()
}

func (r *Repo) GetGraphNode(ctx context.Context, userID, id uuid.UUID) (*GraphNode, error) {
	var n GraphNode
	err := r.db.Read().QueryRow(ctx, graphNodeSelect+` WHERE n.id=$1 AND n.user_id=$2`, id, userID).
		Scan(&n.ID, &n.Kind, &n.ItemID, &n.RefProjectID, &n.Label, &n.Done, &n.Status, &n.Deadline, &n.Note, &n.ImageKeys, &n.X, &n.Y)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, postgres.ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (r *Repo) CreateGraphNode(ctx context.Context, userID uuid.UUID, board *uuid.UUID, kind string, itemID, refProjectID *uuid.UUID, label string, deadline *string, x, y float64) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO gtd_graph_nodes (user_id, board_project_id, kind, item_id, ref_project_id, label, deadline, x, y)
		 VALUES ($1,$2,$3,$4,$5,$6,$7::date,$8,$9) RETURNING id`,
		userID, board, kind, itemID, refProjectID, label, deadline, x, y).Scan(&id)
	return id, err
}

// SetGraphNodeDeadline sets or clears a node's deadline (nil clears it).
func (r *Repo) SetGraphNodeDeadline(ctx context.Context, userID, id uuid.UUID, deadline *string) error {
	ct, err := r.db.Pool.Exec(ctx,
		`UPDATE gtd_graph_nodes SET deadline=$3::date, updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, deadline)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) SetGraphNodeNote(ctx context.Context, userID, id uuid.UUID, note string) error {
	return r.execNode(ctx, `UPDATE gtd_graph_nodes SET note=$3, updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, note)
}

// AddGraphNodeImage appends an image key (idempotent).
func (r *Repo) AddGraphNodeImage(ctx context.Context, userID, id uuid.UUID, key string) error {
	return r.execNode(ctx,
		`UPDATE gtd_graph_nodes SET image_keys = (
			SELECT array_agg(DISTINCT k) FROM unnest(array_append(image_keys, $3)) AS k
		 ), updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, key)
}

// RemoveGraphNodeImage drops one image key from the node.
func (r *Repo) RemoveGraphNodeImage(ctx context.Context, userID, id uuid.UUID, key string) error {
	return r.execNode(ctx,
		`UPDATE gtd_graph_nodes SET image_keys=array_remove(image_keys, $3), updated_at=now()
		 WHERE id=$1 AND user_id=$2`, id, userID, key)
}

func (r *Repo) execNode(ctx context.Context, q string, args ...any) error {
	ct, err := r.db.Pool.Exec(ctx, q, args...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) GetSettings(ctx context.Context, userID uuid.UUID) (Settings, error) {
	var s Settings
	err := r.db.Read().QueryRow(ctx, `SELECT soon_days, grace_days FROM gtd_graph_settings WHERE user_id=$1`, userID).
		Scan(&s.SoonDays, &s.GraceDays)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultSettings(), nil
		}
		return Settings{}, err
	}
	return s, nil
}

func (r *Repo) UpsertSettings(ctx context.Context, userID uuid.UUID, s Settings) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO gtd_graph_settings (user_id, soon_days, grace_days) VALUES ($1,$2,$3)
		 ON CONFLICT (user_id) DO UPDATE SET soon_days=$2, grace_days=$3, updated_at=now()`,
		userID, s.SoonDays, s.GraceDays)
	return err
}

func (r *Repo) MoveGraphNode(ctx context.Context, userID, id uuid.UUID, x, y float64) error {
	ct, err := r.db.Pool.Exec(ctx,
		`UPDATE gtd_graph_nodes SET x=$3, y=$4, updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, x, y)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) SetGraphNodeLabel(ctx context.Context, userID, id uuid.UUID, label string) error {
	ct, err := r.db.Pool.Exec(ctx,
		`UPDATE gtd_graph_nodes SET label=$3, updated_at=now() WHERE id=$1 AND user_id=$2`, id, userID, label)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

func (r *Repo) DeleteGraphNode(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM gtd_graph_nodes WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// CreateGraphEdge links two nodes on the same board (idempotent).
func (r *Repo) CreateGraphEdge(ctx context.Context, userID uuid.UUID, board *uuid.UUID, source, target uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO gtd_graph_edges (user_id, board_project_id, source_id, target_id)
		 SELECT $1,$2,$3,$4
		 WHERE EXISTS (SELECT 1 FROM gtd_graph_nodes WHERE id=$3 AND user_id=$1 AND board_project_id IS NOT DISTINCT FROM $2)
		   AND EXISTS (SELECT 1 FROM gtd_graph_nodes WHERE id=$4 AND user_id=$1 AND board_project_id IS NOT DISTINCT FROM $2)
		 ON CONFLICT (source_id, target_id) DO NOTHING
		 RETURNING id`, userID, board, source, target).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either the edge already exists (conflict) or a node didn't validate.
		e := r.db.Read().QueryRow(ctx,
			`SELECT id FROM gtd_graph_edges WHERE user_id=$1 AND source_id=$2 AND target_id=$3`, userID, source, target)
		if err2 := e.Scan(&id); err2 == nil {
			return id, nil
		}
		return uuid.Nil, postgres.ErrNotFound
	}
	return id, err
}

func (r *Repo) DeleteGraphEdge(ctx context.Context, userID, id uuid.UUID) error {
	ct, err := r.db.Pool.Exec(ctx, `DELETE FROM gtd_graph_edges WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return postgres.ErrNotFound
	}
	return nil
}

// ---------- service ----------

// AddGraphNode creates a node, creating the underlying task/sub-project when one
// isn't linked. board = nil is the root board.
func (s *Service) AddGraphNode(ctx context.Context, userID uuid.UUID, board *uuid.UUID, kind, title string, existingItem, existingProject *uuid.UUID, deadline *string, x, y float64) (*GraphNode, error) {
	var itemID, refProject *uuid.UUID
	label := ""
	switch kind {
	case "task":
		if existingItem != nil {
			itemID = existingItem
		} else {
			it, err := s.repo.CreateItem(ctx, userID, ItemInput{Title: title, Bucket: "next", ProjectID: board})
			if err != nil {
				return nil, err
			}
			itemID = &it.ID
		}
	case "project":
		if existingProject != nil {
			refProject = existingProject
		} else {
			p, err := s.repo.CreateProject(ctx, userID, ProjectInput{Title: title, Status: "active"})
			if err != nil {
				return nil, err
			}
			refProject = &p.ID
		}
	case "note":
		label = title
	default:
		return nil, errBadKind
	}
	id, err := s.repo.CreateGraphNode(ctx, userID, board, kind, itemID, refProject, label, deadline, x, y)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.GetGraphNode(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	st, _ := s.repo.GetSettings(ctx, userID)
	s.colorLeaf(n, time.Now(), st)
	return n, nil
}

func (s *Service) SetNodeDeadline(ctx context.Context, userID, id uuid.UUID, deadline *string) error {
	return s.repo.SetGraphNodeDeadline(ctx, userID, id, deadline)
}
func (s *Service) GetSettings(ctx context.Context, userID uuid.UUID) (Settings, error) {
	return s.repo.GetSettings(ctx, userID)
}
func (s *Service) SetSettings(ctx context.Context, userID uuid.UUID, in Settings) error {
	return s.repo.UpsertSettings(ctx, userID, in)
}

// ---------- colour logic ----------

// colorLeaf sets a task/note node's colour from its deadline. Project nodes are
// coloured by aggregation (see BoardView).
func (s *Service) colorLeaf(n *GraphNode, now time.Time, st Settings) {
	n.Color = leafColor(n.Deadline, now, st)
}

func leafColor(deadline *string, now time.Time, st Settings) string {
	if deadline == nil || *deadline == "" {
		return "green"
	}
	d, err := time.Parse("2006-01-02", *deadline)
	if err != nil {
		return "green"
	}
	days := dayDiff(now, d) // deadline − today, in whole days
	switch {
	case days < -st.GraceDays:
		return "red"
	case days <= st.SoonDays:
		return "yellow"
	default:
		return "green"
	}
}

func dayDiff(now, deadline time.Time) int {
	a := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

func worst(a, b string) string {
	rank := map[string]int{"green": 1, "yellow": 2, "red": 3}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

func maxDate(cur *string, cand *string) *string {
	if cand == nil || *cand == "" {
		return cur
	}
	if cur == nil || *cand > *cur {
		return cand
	}
	return cur
}

// BoardView loads a board, colours its nodes (recursing into sub-project nodes
// so their colour reflects their own worst node) and returns a summary. visited
// guards against project-of-project cycles.
func (s *Service) BoardView(ctx context.Context, userID uuid.UUID, board *uuid.UUID, st Settings, now time.Time, visited map[uuid.UUID]bool) ([]GraphNode, []GraphEdge, Summary, error) {
	nodes, edges, err := s.repo.ListGraph(ctx, userID, board)
	if err != nil {
		return nil, nil, Summary{}, err
	}
	sum := Summary{Color: "green"}
	for i := range nodes {
		n := &nodes[i]
		if n.Kind == "project" && n.RefProjectID != nil {
			if visited[*n.RefProjectID] {
				n.Color = "green" // cycle guard
			} else {
				visited[*n.RefProjectID] = true
				_, _, subSum, err := s.BoardView(ctx, userID, n.RefProjectID, st, now, visited)
				delete(visited, *n.RefProjectID)
				if err != nil {
					return nil, nil, Summary{}, err
				}
				n.Color = subSum.Color
				sum.Projected = maxDate(sum.Projected, subSum.Projected)
			}
		} else {
			n.Color = leafColor(n.Deadline, now, st)
			sum.Projected = maxDate(sum.Projected, n.Deadline)
		}
		n.ImageURLs = s.imageURLs(ctx, n.ImageKeys)
		switch n.Color {
		case "red":
			sum.Red++
		case "yellow":
			sum.Yellow++
		default:
			sum.Green++
		}
		sum.Color = worst(sum.Color, n.Color)
	}
	return nodes, edges, sum, nil
}

func (s *Service) SetNodeNote(ctx context.Context, userID, id uuid.UUID, note string) error {
	return s.repo.SetGraphNodeNote(ctx, userID, id, note)
}
func (s *Service) AddNodeImage(ctx context.Context, userID, id uuid.UUID, key string) error {
	return s.repo.AddGraphNodeImage(ctx, userID, id, key)
}
func (s *Service) RemoveNodeImage(ctx context.Context, userID, id uuid.UUID, key string) error {
	return s.repo.RemoveGraphNodeImage(ctx, userID, id, key)
}

// Boards returns every graph (root + each project) with its colour summary.
func (s *Service) Boards(ctx context.Context, userID uuid.UUID) ([]BoardInfo, error) {
	st, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := []BoardInfo{}

	_, _, rootSum, err := s.BoardView(ctx, userID, nil, st, now, map[uuid.UUID]bool{})
	if err != nil {
		return nil, err
	}
	out = append(out, BoardInfo{ProjectID: nil, Title: "Все проекты", Summary: rootSum})

	projects, err := s.repo.ListProjects(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range projects {
		p := projects[i]
		_, _, sum, err := s.BoardView(ctx, userID, &p.ID, st, now, map[uuid.UUID]bool{p.ID: true})
		if err != nil {
			return nil, err
		}
		pid := p.ID
		out = append(out, BoardInfo{ProjectID: &pid, Title: p.Title, Summary: sum})
	}
	return out, nil
}

var errBadKind = errors.New("invalid node kind")

func (s *Service) Graph(ctx context.Context, userID uuid.UUID, board *uuid.UUID) ([]GraphNode, []GraphEdge, error) {
	return s.repo.ListGraph(ctx, userID, board)
}
func (s *Service) MoveNode(ctx context.Context, userID, id uuid.UUID, x, y float64) error {
	return s.repo.MoveGraphNode(ctx, userID, id, x, y)
}
func (s *Service) RelabelNode(ctx context.Context, userID, id uuid.UUID, label string) error {
	return s.repo.SetGraphNodeLabel(ctx, userID, id, label)
}
func (s *Service) DeleteNode(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteGraphNode(ctx, userID, id)
}
func (s *Service) AddEdge(ctx context.Context, userID uuid.UUID, board *uuid.UUID, source, target uuid.UUID) (uuid.UUID, error) {
	return s.repo.CreateGraphEdge(ctx, userID, board, source, target)
}
func (s *Service) DeleteEdge(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteGraphEdge(ctx, userID, id)
}

// ---------- handler ----------

type graphNodeDTO struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	ItemID       *string  `json:"itemId"`
	RefProjectID *string  `json:"refProjectId"`
	Label        string   `json:"label"`
	Done         bool     `json:"done"`
	Status       string   `json:"status"`
	Deadline     *string  `json:"deadline"`
	Color        string   `json:"color"`
	Note         string   `json:"note"`
	ImageKeys    []string `json:"imageKeys"`
	ImageURLs    []string `json:"imageUrls"`
	X            float64  `json:"x"`
	Y            float64  `json:"y"`
}

type summaryDTO struct {
	Red       int     `json:"red"`
	Yellow    int     `json:"yellow"`
	Green     int     `json:"green"`
	Color     string  `json:"color"`
	Projected *string `json:"projectedCompletion"`
}

func toSummaryDTO(s Summary) summaryDTO {
	return summaryDTO{Red: s.Red, Yellow: s.Yellow, Green: s.Green, Color: s.Color, Projected: s.Projected}
}

type graphEdgeDTO struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func toGraphNodeDTO(n GraphNode) graphNodeDTO {
	color := n.Color
	if color == "" {
		color = "green"
	}
	urls := n.ImageURLs
	if urls == nil {
		urls = []string{}
	}
	keys := n.ImageKeys
	if keys == nil {
		keys = []string{}
	}
	d := graphNodeDTO{ID: n.ID.String(), Kind: n.Kind, Label: n.Label, Done: n.Done, Status: n.Status, Deadline: n.Deadline, Color: color, Note: n.Note, ImageKeys: keys, ImageURLs: urls, X: n.X, Y: n.Y}
	if n.ItemID != nil {
		s := n.ItemID.String()
		d.ItemID = &s
	}
	if n.RefProjectID != nil {
		s := n.RefProjectID.String()
		d.RefProjectID = &s
	}
	return d
}

// parseBoard reads the board id from a string ("" → root board).
func parseBoard(s string) (*uuid.UUID, *httpx.APIError) {
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, httpx.ErrBadRequest("invalid project id")
	}
	return &id, nil
}

func (h *Handler) getGraph(w http.ResponseWriter, r *http.Request) {
	board, apiErr := parseBoard(r.URL.Query().Get("project"))
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	userID := auth.UserID(r.Context())
	st, err := h.svc.GetSettings(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	visited := map[uuid.UUID]bool{}
	if board != nil {
		visited[*board] = true
	}
	nodes, edges, sum, err := h.svc.BoardView(r.Context(), userID, board, st, time.Now(), visited)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	nd := make([]graphNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		nd = append(nd, toGraphNodeDTO(n))
	}
	ed := make([]graphEdgeDTO, 0, len(edges))
	for _, e := range edges {
		ed = append(ed, graphEdgeDTO{ID: e.ID.String(), Source: e.Source.String(), Target: e.Target.String()})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"nodes": nd, "edges": ed, "summary": toSummaryDTO(sum)})
}

type boardInfoDTO struct {
	ProjectID *string    `json:"projectId"`
	Title     string     `json:"title"`
	Summary   summaryDTO `json:"summary"`
}

func (h *Handler) getBoards(w http.ResponseWriter, r *http.Request) {
	boards, err := h.svc.Boards(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	out := make([]boardInfoDTO, 0, len(boards))
	for _, b := range boards {
		var pid *string
		if b.ProjectID != nil {
			s := b.ProjectID.String()
			pid = &s
		}
		out = append(out, boardInfoDTO{ProjectID: pid, Title: b.Title, Summary: toSummaryDTO(b.Summary)})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": out})
}

type settingsDTO struct {
	SoonDays  int `json:"soonDays"`
	GraceDays int `json:"graceDays"`
}

func (h *Handler) getGraphSettings(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.GetSettings(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, settingsDTO{SoonDays: st.SoonDays, GraceDays: st.GraceDays})
}

func (h *Handler) putGraphSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsDTO
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	v := validate.New()
	v.Check(req.SoonDays >= 0 && req.SoonDays <= 365, "soonDays", "0..365")
	v.Check(req.GraceDays >= 0 && req.GraceDays <= 365, "graceDays", "0..365")
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	if err := h.svc.SetSettings(r.Context(), auth.UserID(r.Context()), Settings{SoonDays: req.SoonDays, GraceDays: req.GraceDays}); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, req)
}

type addNodeRequest struct {
	Project      string  `json:"project"` // board; "" = root
	Kind         string  `json:"kind"`
	Title        string  `json:"title"`
	ItemID       *string `json:"itemId"`
	RefProjectID *string `json:"refProjectId"`
	Deadline     *string `json:"deadline"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
}

func (h *Handler) createGraphNode(w http.ResponseWriter, r *http.Request) {
	var req addNodeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	board, apiErr := parseBoard(req.Project)
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	v := validate.New()
	v.Check(req.Kind == "task" || req.Kind == "project" || req.Kind == "note", "kind", "invalid kind")
	linking := (req.ItemID != nil && *req.ItemID != "") || (req.RefProjectID != nil && *req.RefProjectID != "")
	if !linking {
		v.Required("title", req.Title)
		v.MaxLen("title", req.Title, 500)
	}
	if !v.Valid() {
		httpx.Error(w, r, httpx.ValidationError(v.Errors))
		return
	}
	existingItem, err1 := parseOptUUID(req.ItemID)
	existingProject, err2 := parseOptUUID(req.RefProjectID)
	if err1 != nil || err2 != nil {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid linked id"))
		return
	}
	var deadline *string
	if req.Deadline != nil && *req.Deadline != "" {
		deadline = req.Deadline
	}
	n, err := h.svc.AddGraphNode(r.Context(), auth.UserID(r.Context()), board, req.Kind, req.Title, existingItem, existingProject, deadline, req.X, req.Y)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toGraphNodeDTO(*n))
}

type updateNodeRequest struct {
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Label    *string  `json:"label"`
	Note     *string  `json:"note"`
	Deadline *string  `json:"deadline"` // "" clears
}

func (h *Handler) updateGraphNode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req updateNodeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	userID := auth.UserID(r.Context())
	if req.X != nil && req.Y != nil {
		if err := h.svc.MoveNode(r.Context(), userID, id, *req.X, *req.Y); err != nil {
			renderErr(w, r, err, "node not found")
			return
		}
	}
	if req.Label != nil {
		if err := h.svc.RelabelNode(r.Context(), userID, id, *req.Label); err != nil {
			renderErr(w, r, err, "node not found")
			return
		}
	}
	if req.Note != nil {
		if err := h.svc.SetNodeNote(r.Context(), userID, id, *req.Note); err != nil {
			renderErr(w, r, err, "node not found")
			return
		}
	}
	if req.Deadline != nil {
		var d *string
		if *req.Deadline != "" {
			d = req.Deadline
		}
		if err := h.svc.SetNodeDeadline(r.Context(), userID, id, d); err != nil {
			renderErr(w, r, err, "node not found")
			return
		}
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type imageUploadRequest struct {
	ContentType string `json:"contentType"`
}

func (h *Handler) graphImageUploadURL(w http.ResponseWriter, r *http.Request) {
	var req imageUploadRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	url, key, err := h.svc.PrepareImageUpload(r.Context(), req.ContentType)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedMedia):
			httpx.Error(w, r, httpx.ErrBadRequest("поддерживаются JPEG, PNG, WEBP"))
		case errors.Is(err, ErrStorageDisabled):
			httpx.Error(w, r, httpx.ErrBadRequest("загрузка изображений отключена"))
		default:
			httpx.Error(w, r, err)
		}
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"uploadUrl": url, "key": key})
}

type nodeImageRequest struct {
	Key string `json:"key"`
}

func (h *Handler) addGraphNodeImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req nodeImageRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Key == "" {
		httpx.Error(w, r, httpx.ErrBadRequest("key required"))
		return
	}
	if err := h.svc.AddNodeImage(r.Context(), auth.UserID(r.Context()), id, req.Key); err != nil {
		renderErr(w, r, err, "node not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) removeGraphNodeImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req nodeImageRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.svc.RemoveNodeImage(r.Context(), auth.UserID(r.Context()), id, req.Key); err != nil {
		renderErr(w, r, err, "node not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func (h *Handler) deleteGraphNode(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteNode(r.Context(), auth.UserID(r.Context()), id); err != nil {
		renderErr(w, r, err, "node not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

type addEdgeRequest struct {
	Project string `json:"project"`
	Source  string `json:"source"`
	Target  string `json:"target"`
}

func (h *Handler) createGraphEdge(w http.ResponseWriter, r *http.Request) {
	var req addEdgeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	board, apiErr := parseBoard(req.Project)
	if apiErr != nil {
		httpx.Error(w, r, apiErr)
		return
	}
	source, err1 := uuid.Parse(req.Source)
	target, err2 := uuid.Parse(req.Target)
	if err1 != nil || err2 != nil || source == target {
		httpx.Error(w, r, httpx.ErrBadRequest("invalid edge"))
		return
	}
	id, err := h.svc.AddEdge(r.Context(), auth.UserID(r.Context()), board, source, target)
	if err != nil {
		renderErr(w, r, err, "nodes not found")
		return
	}
	httpx.JSON(w, http.StatusCreated, graphEdgeDTO{ID: id.String(), Source: req.Source, Target: req.Target})
}

func (h *Handler) deleteGraphEdge(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteEdge(r.Context(), auth.UserID(r.Context()), id); err != nil {
		renderErr(w, r, err, "edge not found")
		return
	}
	httpx.JSON(w, http.StatusNoContent, nil)
}

func parseOptUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
