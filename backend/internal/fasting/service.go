package fasting

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
)

// ErrNoActiveFast means the user tried to stop/adjust a fast that isn't running.
var ErrNoActiveFast = errors.New("no active fast")

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

func hoursToDur(h float64) time.Duration { return time.Duration(h * float64(time.Hour)) }

// State computes the live phase (ongoing fast, following eating window, or idle).
func (s *Service) State(ctx context.Context, userID uuid.UUID, now time.Time) (*State, error) {
	set, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	st := &State{Phase: "idle", FastingHours: set.FastingHours, EatingHours: set.EatingHours, ServerNow: now}

	active, err := s.repo.GetActive(ctx, userID)
	switch {
	case err == nil:
		st.Phase = "fasting"
		id := active.ID
		st.ActiveID = &id
		st.GoalHours = active.GoalHours
		start := active.StartedAt
		end := active.StartedAt.Add(hoursToDur(active.GoalHours))
		st.PhaseStartAt = &start
		st.PhaseEndAt = &end
		st.Overrun = now.After(end)
	case errors.Is(err, postgres.ErrNotFound):
		last, lerr := s.repo.GetLast(ctx, userID)
		if lerr == nil && last.EndedAt != nil {
			st.Phase = "eating"
			st.GoalHours = set.EatingHours
			start := *last.EndedAt
			end := last.EndedAt.Add(hoursToDur(set.EatingHours))
			st.PhaseStartAt = &start
			st.PhaseEndAt = &end
			st.Overrun = now.After(end)
		} else if lerr != nil && !errors.Is(lerr, postgres.ErrNotFound) {
			return nil, lerr
		}
	default:
		return nil, err
	}

	stats, err := s.repo.Stats(ctx, userID)
	if err != nil {
		return nil, err
	}
	st.Stats = stats

	sched, err := s.repo.GetSchedule(ctx, userID)
	if err != nil {
		return nil, err
	}
	st.Schedule = sched
	return st, nil
}

// Start begins a fast (idempotent: returns the current state if already fasting).
func (s *Service) Start(ctx context.Context, userID uuid.UUID, startedAt *time.Time) (*State, error) {
	now := time.Now()
	if _, err := s.repo.GetActive(ctx, userID); err == nil {
		return s.State(ctx, userID, now) // already fasting
	} else if !errors.Is(err, postgres.ErrNotFound) {
		return nil, err
	}
	set, err := s.repo.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	start := now
	if startedAt != nil {
		start = *startedAt
		if start.After(now) {
			start = now
		}
	}
	if _, err := s.repo.Create(ctx, userID, start, set.FastingHours); err != nil {
		return nil, err
	}
	return s.State(ctx, userID, time.Now())
}

// Stop ends the ongoing fast.
func (s *Service) Stop(ctx context.Context, userID uuid.UUID, endedAt *time.Time) (*State, error) {
	now := time.Now()
	active, err := s.repo.GetActive(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNoActiveFast
		}
		return nil, err
	}
	end := now
	if endedAt != nil {
		end = *endedAt
	}
	if end.After(now) {
		end = now
	}
	if end.Before(active.StartedAt) {
		end = active.StartedAt
	}
	if _, err := s.repo.End(ctx, active.ID, userID, end); err != nil {
		return nil, err
	}
	return s.State(ctx, userID, time.Now())
}

// SetActiveStart adjusts the current fast's start time (e.g. "I started 2h ago").
func (s *Service) SetActiveStart(ctx context.Context, userID uuid.UUID, startedAt time.Time) (*State, error) {
	now := time.Now()
	if startedAt.After(now) {
		startedAt = now
	}
	if _, err := s.repo.UpdateActiveStart(ctx, userID, startedAt); err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrNoActiveFast
		}
		return nil, err
	}
	return s.State(ctx, userID, time.Now())
}

func (s *Service) SetSettings(ctx context.Context, userID uuid.UUID, in Settings) (*State, error) {
	if err := s.repo.UpsertSettings(ctx, userID, in); err != nil {
		return nil, err
	}
	return s.State(ctx, userID, time.Now())
}

// SetSchedule validates and stores the daily schedule, then returns fresh state.
func (s *Service) SetSchedule(ctx context.Context, userID uuid.UUID, in Schedule) (*State, error) {
	if in.Timezone == "" {
		in.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		in.Timezone = "UTC" // unknown zone from the client → fall back rather than reject
	}
	if in.EatStartHour < 0 || in.EatStartHour > 23 {
		in.EatStartHour = 12
	}
	if in.EatStartMin < 0 || in.EatStartMin > 59 {
		in.EatStartMin = 0
	}
	if err := s.repo.UpsertSchedule(ctx, userID, in); err != nil {
		return nil, err
	}
	return s.State(ctx, userID, time.Now())
}

func (s *Service) History(ctx context.Context, userID uuid.UUID, limit int) ([]Session, error) {
	return s.repo.List(ctx, userID, limit)
}

func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, id, userID)
}
