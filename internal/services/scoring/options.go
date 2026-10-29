package scoring

import (
	"context"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

// Dimensions is the fixed set of dimensions the bank's options are grouped
// into, with the kind and stances the frontend and validation both need.
var Dimensions = []dto.DimensionSpec{
	{Key: dto.DimensionTech, Kind: "pair", Stances: allStances, Weight: 2, Saturation: 3},
	{Key: dto.DimensionRole, Kind: "pair", Stances: allStances, Weight: 3, Saturation: 1},
	{Key: dto.DimensionDomain, Kind: "pair", Stances: allStances, Weight: 2, Saturation: 1},
	{Key: dto.DimensionSeniority, Kind: ladderKind, Stances: []dto.Stance{dto.StanceNice}, Weight: 3, Saturation: 1},
	{Key: dto.DimensionWork, Kind: "multi", Stances: allStances, Weight: 2, Saturation: 1},
	{Key: dto.DimensionStage, Kind: "multi", Stances: allStances, Weight: 1, Saturation: 1},
	{Key: dto.DimensionSize, Kind: "multi", Stances: allStances, Weight: 1, Saturation: 1},
	{Key: dto.DimensionEmployment, Kind: "multi", Stances: allStances, Weight: 2, Saturation: 1},
}

var allStances = []dto.Stance{dto.StanceNever, dto.StanceAvoid, dto.StanceOK, dto.StanceNice}

var dimensionSpecs = func() map[dto.Dimension]dto.DimensionSpec {
	specs := make(map[dto.Dimension]dto.DimensionSpec, len(Dimensions))
	for _, d := range Dimensions {
		specs[d.Key] = d
	}
	return specs
}()

const (
	ladderKind      = "ladder"
	maxLadderWeight = 100
)

var ladderLevels = map[string]int{
	"seniority:junior":         1,
	"seniority:mid":            2,
	"seniority:senior":         3,
	"seniority:lead_staff":     4,
	"seniority:principal_head": 5,
}

func (s *Service) Options(ctx context.Context, userID string) (dto.ScoringOptionsView, error) {
	bk, err := s.loadBank(ctx, userID)
	if err != nil {
		return dto.ScoringOptionsView{}, err
	}
	out := make([]dto.ScoringOption, 0, len(bk.live))
	custom := make([]dto.CustomOption, 0)
	for _, o := range bk.live {
		if o.OwnerID != "" {
			custom = append(custom, dto.CustomOption{ID: o.ID, Dimension: o.Dimension, Label: o.Label, Question: o.Question})
			continue
		}
		out = append(out, dto.ScoringOption{ID: o.ID, Dimension: o.Dimension, Label: o.Label, Level: ladderLevels[o.ID]})
	}
	return dto.ScoringOptionsView{Dimensions: Dimensions, Options: out, Custom: custom, CustomCap: maxCustomOptions}, nil
}
