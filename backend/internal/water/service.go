package water

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/kabanos/backend/internal/postgres"
	"github.com/kabanos/backend/internal/timex"
)

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// goalML returns the user's goal, or the default when none is configured.
func (s *Service) goalML(ctx context.Context, userID uuid.UUID) (int, error) {
	g, err := s.repo.GetGoal(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return DefaultGoalML, nil
		}
		return 0, err
	}
	return g.DailyML, nil
}

// DaySummary assembles everything the UI needs for one day.
func (s *Service) DaySummary(ctx context.Context, userID uuid.UUID, date, tz string) (*DaySummary, error) {
	loc := timex.Location(tz)
	if date == "" {
		date = timex.Today(loc)
	}
	from, to, err := timex.DayRange(date, loc)
	if err != nil {
		return nil, err
	}

	goal, err := s.goalML(ctx, userID)
	if err != nil {
		return nil, err
	}
	intakes, err := s.repo.ListIntakesBetween(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	consumed := 0
	for _, in := range intakes {
		consumed += in.AmountML
	}

	remaining := goal - consumed
	if remaining < 0 {
		remaining = 0
	}
	percent := 0.0
	if goal > 0 {
		percent = math.Round(float64(consumed)/float64(goal)*1000) / 10 // one decimal
	}
	return &DaySummary{
		Date:        date,
		GoalML:      goal,
		ConsumedML:  consumed,
		RemainingML: remaining,
		Percent:     percent,
		Intakes:     intakes,
	}, nil
}

func (s *Service) SetGoal(ctx context.Context, userID uuid.UUID, dailyML int) (*Goal, error) {
	return s.repo.UpsertGoal(ctx, userID, dailyML)
}

func (s *Service) GetGoalOrDefault(ctx context.Context, userID uuid.UUID) (*Goal, error) {
	g, err := s.repo.GetGoal(ctx, userID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return &Goal{UserID: userID, DailyML: DefaultGoalML, UpdatedAt: time.Time{}}, nil
		}
		return nil, err
	}
	return g, nil
}

// AddIntake records a pour. consumedAt defaults to now when the client omits it.
func (s *Service) AddIntake(ctx context.Context, userID uuid.UUID, amountML int, source string, consumedAt *time.Time) (*Intake, error) {
	when := time.Now()
	if consumedAt != nil {
		when = *consumedAt
	}
	return s.repo.AddIntake(ctx, userID, amountML, source, when)
}

func (s *Service) DeleteIntake(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteIntake(ctx, userID, id)
}

// History returns per-day totals for the inclusive [fromDate, toDate] range,
// filling missing days with zero so the chart is continuous.
func (s *Service) History(ctx context.Context, userID uuid.UUID, fromDate, toDate, tz string) ([]DayTotal, int, error) {
	loc := timex.Location(tz)
	from, to, err := timex.RangeBounds(fromDate, toDate, loc)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.repo.DailyTotals(ctx, userID, from, to, loc.String())
	if err != nil {
		return nil, 0, err
	}
	byDay := make(map[string]int, len(rows))
	for _, d := range rows {
		byDay[d.Date] = d.TotalML
	}

	goal, err := s.goalML(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// Emit a continuous series across the requested range.
	out := []DayTotal{}
	for cur := from; cur.Before(to); cur = cur.AddDate(0, 0, 1) {
		key := cur.Format("2006-01-02")
		out = append(out, DayTotal{Date: key, TotalML: byDay[key]})
	}
	return out, goal, nil
}
