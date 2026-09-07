package pressure

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// AddEntry records a measurement. measuredAt defaults to now when nil.
func (s *Service) AddEntry(ctx context.Context, userID uuid.UUID, systolic, diastolic int, pulse *int, note string, measuredAt *time.Time) (*Entry, error) {
	at := time.Now()
	if measuredAt != nil {
		at = *measuredAt
	}
	return s.repo.Insert(ctx, userID, systolic, diastolic, pulse, note, at)
}

func (s *Service) DeleteEntry(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.DeleteEntry(ctx, userID, id)
}

// Summary builds the diary view: the recent series, per-metric averages, and the
// latest reading with its category.
func (s *Service) Summary(ctx context.Context, userID uuid.UUID, limit int) (*Summary, error) {
	series, err := s.repo.List(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	sum := &Summary{Series: series, Count: len(series), Averages: average(series)}

	latest, err := s.repo.Latest(ctx, userID)
	if err == nil {
		sum.Latest = latest
		sum.Category = Category(latest.Systolic, latest.Diastolic)
	}
	return sum, nil
}

// average computes the mean systolic/diastolic (over all entries) and pulse
// (over entries that recorded one), rounded to one decimal.
func average(series []Entry) Averages {
	if len(series) == 0 {
		return Averages{}
	}
	var sysSum, diaSum, pulseSum float64
	var pulseN int
	for _, e := range series {
		sysSum += float64(e.Systolic)
		diaSum += float64(e.Diastolic)
		if e.Pulse != nil {
			pulseSum += float64(*e.Pulse)
			pulseN++
		}
	}
	n := float64(len(series))
	sys := round1(sysSum / n)
	dia := round1(diaSum / n)
	avg := Averages{Systolic: &sys, Diastolic: &dia}
	if pulseN > 0 {
		p := round1(pulseSum / float64(pulseN))
		avg.Pulse = &p
	}
	return avg
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
