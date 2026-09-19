package habits

import (
	"context"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/reminders"
)

type Service struct {
	repo *Repo
	rem  *reminders.Service
}

func NewService(repo *Repo, rem *reminders.Service) *Service {
	return &Service{repo: repo, rem: rem}
}

// --- habits ---

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Habit, error) {
	return s.repo.List(ctx, userID)
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
