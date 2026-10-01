package wttj

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

type companyProfileEntity struct {
	Mission         string      `json:"mission"`
	HQ              string      `json:"parsedHqAddress"`
	GlassdoorRating string      `json:"glassdoorRating"`
	Growth          *float64    `json:"yearEmployeeGrowthPercentage"`
	Sectors         []apolloRef `json:"sectorTags"`
	Size            *apolloRef  `json:"size"`
	VisaCountries   []apolloRef `json:"visaSponsorshipCountries"`
}

// apolloArgs holds the two company fields whose Apollo cache keys carry query arguments,
// which a struct tag cannot name.
type apolloArgs struct {
	TotalFunding  *struct{ Amount string } `json:"-"`
	FundingRounds []apolloRef              `json:"-"`
}

const (
	totalFundingKey  = `totalFunding({"currency":"USD"})`
	fundingRoundsKey = `fundingRounds({"currency":"USD"})`
)

func parseApolloArgs(state map[string]json.RawMessage, companyKey string) (apolloArgs, error) {
	fields, err := entity[map[string]json.RawMessage](state, companyKey)
	if err != nil {
		return apolloArgs{}, err
	}
	var a apolloArgs
	if raw, ok := fields[totalFundingKey]; ok {
		if err := json.Unmarshal(raw, &a.TotalFunding); err != nil {
			return apolloArgs{}, fmt.Errorf("wttj: decode total funding: %w", err)
		}
	}
	if raw, ok := fields[fundingRoundsKey]; ok {
		if err := json.Unmarshal(raw, &a.FundingRounds); err != nil {
			return apolloArgs{}, fmt.Errorf("wttj: decode funding rounds: %w", err)
		}
	}
	return a, nil
}

type valueEntity struct {
	Value string `json:"value"`
}

type visaEntity struct {
	Location   string `json:"location"`
	OffersVisa bool   `json:"offersVisa"`
}

func parseProfile(state map[string]json.RawMessage, companyKey string) *dto.CompanyProfile {
	p, err := buildProfile(state, companyKey)
	if err != nil {
		slog.Warn("wttj profile unparseable", slog.Any(logger.KeyErr, err))
		return nil
	}
	return p
}

func entity[T any](state map[string]json.RawMessage, key string) (T, error) {
	var v T
	if err := json.Unmarshal(state[key], &v); err != nil {
		return v, fmt.Errorf("wttj: decode %s: %w", key, err)
	}
	return v, nil
}

func buildProfile(state map[string]json.RawMessage, companyKey string) (*dto.CompanyProfile, error) {
	c, err := entity[companyProfileEntity](state, companyKey)
	if err != nil {
		return nil, err
	}
	args, err := parseApolloArgs(state, companyKey)
	if err != nil {
		return nil, err
	}
	p := &dto.CompanyProfile{
		HQ:            c.HQ,
		Mission:       c.Mission,
		Glassdoor:     c.GlassdoorRating,
		FundingRounds: len(args.FundingRounds),
	}
	for _, ref := range c.Sectors {
		tag, err := entity[valueEntity](state, ref.Ref)
		if err != nil {
			return nil, err
		}
		p.Sectors = append(p.Sectors, tag.Value)
	}
	if c.Size != nil {
		size, err := entity[valueEntity](state, c.Size.Ref)
		if err != nil {
			return nil, err
		}
		p.Size = size.Value
	}
	if c.Growth != nil {
		p.Growth = fmt.Sprintf("%+.0f%%", *c.Growth)
	}
	if args.TotalFunding != nil {
		p.FundingTotal = formatUSD(args.TotalFunding.Amount)
	}
	for _, ref := range c.VisaCountries {
		visa, err := entity[visaEntity](state, ref.Ref)
		if err != nil {
			return nil, err
		}
		if visa.Location == "UK" {
			p.UKVisa = "no"
			if visa.OffersVisa {
				p.UKVisa = "yes"
			}
		}
	}
	return p, nil
}

func formatUSD(amount string) string {
	n, err := strconv.ParseFloat(amount, 64)
	switch {
	case err != nil:
		return ""
	case n >= 1e9:
		return fmt.Sprintf("$%.1fB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("$%.1fM", n/1e6)
	default:
		return fmt.Sprintf("$%.0fK", n/1e3)
	}
}
