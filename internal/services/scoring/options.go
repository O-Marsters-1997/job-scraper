package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Dimensions is the fixed set of dimensions the bank's options are grouped
// into, with the kind and stances the frontend and validation both need.
var Dimensions = []dto.DimensionSpec{
	{Key: dto.DimensionTech, Kind: "pair", Stances: []string{"nice", "avoid"}},
	{Key: dto.DimensionRole, Kind: "pair", Stances: []string{"nice", "avoid"}},
	{Key: dto.DimensionDomain, Kind: "pair", Stances: []string{"nice", "avoid", "block"}},
	{Key: dto.DimensionSeniority, Kind: "multi", Stances: []string{"nice"}},
	{Key: dto.DimensionWork, Kind: "multi", Stances: []string{"nice"}},
	{Key: dto.DimensionStage, Kind: "multi", Stances: []string{"nice"}},
}

// Options returns the dimensions and every non-retired bank option, without
// question text.
func (s *Service) Options(ctx context.Context, _ string) (dto.ScoringOptionsView, error) {
	bk, err := s.loadBank(ctx)
	if err != nil {
		return dto.ScoringOptionsView{}, err
	}
	live := bk.live
	out := make([]dto.ScoringOption, len(live))
	for i, o := range live {
		out[i] = dto.ScoringOption{ID: o.ID, Dimension: o.Dimension, Label: o.Label}
	}
	return dto.ScoringOptionsView{Dimensions: Dimensions, Options: out}, nil
}
