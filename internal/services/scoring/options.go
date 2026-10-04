package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Dimensions is the fixed set of dimensions the bank's options are grouped
// into, with the kind and stances the frontend and validation both need.
var Dimensions = []dto.DimensionSpec{
	{Key: dto.DimensionTech, Kind: "pair", Stances: []string{"nice", "ok", "avoid"}, Weight: 2, Saturation: 3},
	{Key: dto.DimensionRole, Kind: "pair", Stances: []string{"nice", "ok", "avoid"}, Weight: 3, Saturation: 1},
	{Key: dto.DimensionDomain, Kind: "pair", Stances: []string{"nice", "ok", "avoid", "block"}, Weight: 2, Saturation: 1},
	{Key: dto.DimensionSeniority, Kind: "multi", Stances: []string{"nice", "ok"}, Gate: true, Weight: 3, Saturation: 1},
	{Key: dto.DimensionWork, Kind: "multi", Stances: []string{"nice", "ok"}, Gate: true, Weight: 1, Saturation: 1},
	{Key: dto.DimensionStage, Kind: "multi", Stances: []string{"nice", "ok"}, Weight: 1, Saturation: 1},
	{Key: dto.DimensionSize, Kind: "multi", Stances: []string{"nice", "ok"}, Weight: 1, Saturation: 1},
	{Key: dto.DimensionEmployment, Kind: "multi", Stances: []string{"nice", "ok"}, Gate: true, Weight: 2, Saturation: 1},
}

var dimensionSpecs = func() map[dto.Dimension]dto.DimensionSpec {
	specs := make(map[dto.Dimension]dto.DimensionSpec, len(Dimensions))
	for _, d := range Dimensions {
		specs[d.Key] = d
	}
	return specs
}()

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
