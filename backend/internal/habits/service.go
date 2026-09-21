package habits

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/reminders"
)

// recentWindow is how many days the dashboard preview strip and health colour
// look back.
const recentWindow = 14

const dayLayout = "2006-01-02"

type Service struct {
	repo *Repo
	rem  *reminders.Service
}

func NewService(repo *Repo, rem *reminders.Service) *Service {
	return &Service{repo: repo, rem: rem}
}

// --- habits ---

// List returns the user's habits enriched with recent check-in health (colour,
// streak, preview strip) and ordered worst-first: red habits float to the top,
// then yellow, then green. `today` is the user's local date (YYYY-MM-DD).
func (s *Service) List(ctx context.Context, userID uuid.UUID, today string) ([]Habit, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	t0, err := time.Parse(dayLayout, today)
	if err != nil {
		t0 = time.Now().UTC()
	}
	since := t0.AddDate(0, 0, -(recentWindow - 1)).Format(dayLayout)
	rows, err := s.repo.RecentCheckins(ctx, userID, since)
	if err != nil {
		return nil, err
	}
	// Group check-ins by habit → day → success.
	byHabit := map[uuid.UUID]map[string]bool{}
	for _, c := range rows {
		m := byHabit[c.HabitID]
		if m == nil {
			m = map[string]bool{}
			byHabit[c.HabitID] = m
		}
		m[c.Day] = c.Success
	}
	for i := range items {
		enrich(&items[i], byHabit[items[i].ID], t0)
	}
	sort.SliceStable(items, func(a, b int) bool {
		ra, rb := colorRank(items[a].Color), colorRank(items[b].Color)
		if ra != rb {
			return ra > rb // red first
		}
		if items[a].Fails != items[b].Fails {
			return items[a].Fails > items[b].Fails
		}
		return items[a].Name < items[b].Name
	})
	return items, nil
}

func colorRank(c string) int {
	switch c {
	case "red":
		return 3
	case "yellow":
		return 2
	default:
		return 1
	}
}

// enrich fills a habit's recent strip, fail/success counts, colour and streak
// from its check-in map over the recent window ending at t0 (today).
func enrich(h *Habit, marks map[string]bool, t0 time.Time) {
	recent := make([]Checkin, 0, recentWindow)
	fails, succ := 0, 0
	for i := recentWindow - 1; i >= 0; i-- {
		day := t0.AddDate(0, 0, -i).Format(dayLayout)
		c := Checkin{Day: day, Status: "none"}
		if ok, present := marks[day]; present {
			c.Success = ok
			if ok {
				c.Status = "success"
				succ++
			} else {
				c.Status = "fail"
				fails++
			}
		}
		recent = append(recent, c)
	}
	h.Recent = recent
	h.Fails = fails
	h.Successes = succ
	switch {
	case fails >= 3:
		h.Color = "red"
	case fails >= 1:
		h.Color = "yellow"
	default:
		h.Color = "green"
	}
	// Current streak: consecutive success days ending today (or yesterday if
	// today isn't marked yet), stopping at the first fail or gap.
	streak := 0
	start := t0
	if _, present := marks[t0.Format(dayLayout)]; !present {
		start = t0.AddDate(0, 0, -1)
	}
	for d := start; ; d = d.AddDate(0, 0, -1) {
		ok, present := marks[d.Format(dayLayout)]
		if !present || !ok {
			break
		}
		streak++
	}
	h.Streak = streak
}
func (s *Service) Create(ctx context.Context, userID uuid.UUID, in Input) (*Habit, error) {
	return s.repo.Create(ctx, userID, in)
}
func (s *Service) Update(ctx context.Context, id, userID uuid.UUID, in Input) (*Habit, error) {
	return s.repo.Update(ctx, id, userID, in)
}
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}

// --- daily check-ins (tracker) ---

func (s *Service) Checkins(ctx context.Context, userID, habitID uuid.UUID, from, to string) (map[string]bool, error) {
	return s.repo.CheckinsRange(ctx, userID, habitID, from, to)
}
func (s *Service) SetCheckin(ctx context.Context, userID, habitID uuid.UUID, day string, success bool) error {
	return s.repo.SetCheckin(ctx, userID, habitID, day, success)
}
func (s *Service) ClearCheckin(ctx context.Context, userID, habitID uuid.UUID, day string) error {
	return s.repo.ClearCheckin(ctx, userID, habitID, day)
}

// --- mini-diary ---

func (s *Service) Logs(ctx context.Context, userID, habitID uuid.UUID) ([]Log, error) {
	return s.repo.ListLogs(ctx, userID, habitID, 0)
}
func (s *Service) AddLog(ctx context.Context, userID, habitID uuid.UUID, note, status string) (*Log, error) {
	return s.repo.AddLog(ctx, userID, habitID, note, status)
}
func (s *Service) DeleteLog(ctx context.Context, userID, logID uuid.UUID) error {
	return s.repo.DeleteLog(ctx, userID, logID)
}

// --- reminders (delivered by the shared worker) ---

// ReminderInput is a habit reminder's schedule and message.
type ReminderInput struct {
	Text     string
	Times    []string
	Days     []int
	Timezone string
}

func (s *Service) ListReminders(ctx context.Context, userID, habitID uuid.UUID) ([]reminders.Reminder, error) {
	return s.rem.ListByHabit(ctx, userID, habitID)
}

// AddReminder attaches a fixed-time push reminder to a habit. The push title is
// the habit's name and the body is the user's text.
func (s *Service) AddReminder(ctx context.Context, userID, habitID uuid.UUID, in ReminderInput) (*reminders.Reminder, error) {
	h, err := s.repo.Get(ctx, habitID, userID)
	if err != nil {
		return nil, err
	}
	id := habitID
	return s.rem.Create(ctx, userID, reminders.Input{
		Title:    h.Name,
		Body:     in.Text,
		URL:      "/habits",
		Mode:     "times",
		Times:    in.Times,
		Days:     in.Days,
		Timezone: in.Timezone,
		Enabled:  true,
		HabitID:  &id,
	})
}

func (s *Service) SetReminderEnabled(ctx context.Context, userID, reminderID uuid.UUID, enabled bool) error {
	return s.rem.SetEnabled(ctx, reminderID, userID, enabled)
}

func (s *Service) DeleteReminder(ctx context.Context, userID, reminderID uuid.UUID) error {
	return s.rem.Delete(ctx, reminderID, userID)
}

// Owns reports whether the habit belongs to the user (guards reminder/log routes).
func (s *Service) Owns(ctx context.Context, userID, habitID uuid.UUID) (bool, error) {
	ok, err := s.repo.Owns(ctx, userID, habitID)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, postgres.ErrNotFound
	}
	return true, nil
}
