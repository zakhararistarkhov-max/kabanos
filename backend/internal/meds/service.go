package meds

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/timex"
)

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// View is a medication with per-day progress plus derived course position.
type View struct {
	Progress
	CourseDay   int    // 1-based index of `today` within the course; 0 before start
	CourseTotal *int   // duration in days (nil = open-ended)
	Status      string // upcoming | active | finished
}

// List returns the user's medications with today's counts and course status.
func (s *Service) List(ctx context.Context, userID uuid.UUID, dateStr, tz string) ([]View, string, error) {
	loc := timex.Location(tz)
	if dateStr == "" {
		dateStr = timex.Today(loc)
	}
	today, err := timex.ParseDate(dateStr, loc)
	if err != nil {
		return nil, "", err
	}
	weekStart := today.AddDate(0, 0, -6).Format("2006-01-02")

	progs, err := s.repo.List(ctx, userID, dateStr, weekStart)
	if err != nil {
		return nil, "", err
	}

	views := make([]View, 0, len(progs))
	for _, p := range progs {
		views = append(views, buildView(p, today))
	}
	return views, dateStr, nil
}

func buildView(p Progress, today time.Time) View {
	day := daysBetween(p.StartDate, today) + 1 // 1-based
	status := "active"
	switch {
	case day <= 0:
		status = "upcoming"
	case p.DurationDays != nil && day > *p.DurationDays:
		status = "finished"
	}
	if day < 0 {
		day = 0
	}
	return View{Progress: p, CourseDay: day, CourseTotal: p.DurationDays, Status: status}
}

// daysBetween returns the number of whole calendar days from a to b, comparing
// by date component only so timezone offsets never shift the count.
func daysBetween(a, b time.Time) int {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	au := time.Date(ay, am, ad, 0, 0, 0, 0, time.UTC)
	bu := time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC)
	return int(bu.Sub(au).Hours() / 24)
}

// HistoryDay is one calendar day of a course with the number of intakes logged.
type HistoryDay struct {
	Date  string
	Count int
}

// History is a course's full intake log plus derived totals, letting the user
// review when and how consistently they took it.
type History struct {
	Medication Medication
	Days       []HistoryDay
	TotalTaken int // total intakes logged across the whole course
	ActiveDays int // distinct days with at least one intake
	FirstDate  string
	LastDate   string
}

// History returns the intake log for a course owned by the user.
func (s *Service) History(ctx context.Context, medID, userID uuid.UUID) (*History, error) {
	med, err := s.repo.Get(ctx, medID, userID)
	if err != nil {
		return nil, err
	}
	days, err := s.repo.IntakeHistory(ctx, medID, userID)
	if err != nil {
		return nil, err
	}
	h := History{Medication: *med, Days: make([]HistoryDay, 0, len(days))}
	for _, d := range days {
		h.Days = append(h.Days, HistoryDay{Date: d.Date.Format(dateLayout), Count: d.Count})
		h.TotalTaken += d.Count
	}
	h.ActiveDays = len(days)
	if len(days) > 0 {
		h.FirstDate = days[0].Date.Format(dateLayout)
		h.LastDate = days[len(days)-1].Date.Format(dateLayout)
	}
	return &h, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (*Medication, error) {
	return s.repo.Create(ctx, userID, in)
}

func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Medication, error) {
	return s.repo.Update(ctx, id, userID, in)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

// TakeIntake logs one intake for the user-local day derived from tz.
func (s *Service) TakeIntake(ctx context.Context, medID, userID uuid.UUID, tz string) error {
	return s.repo.AddIntake(ctx, medID, userID, timex.Today(timex.Location(tz)))
}

// UndoIntake removes the most recent intake for the user-local day.
func (s *Service) UndoIntake(ctx context.Context, medID, userID uuid.UUID, tz string) error {
	return s.repo.UndoIntake(ctx, medID, userID, timex.Today(timex.Location(tz)))
}
