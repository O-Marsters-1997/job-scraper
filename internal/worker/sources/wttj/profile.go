package wttj

import (
	"fmt"
	"strconv"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

func (c apiCompany) profile() *dto.CompanyProfile {
	p := &dto.CompanyProfile{
		HQ:            c.HQ,
		Mission:       c.Mission,
		Glassdoor:     c.Glassdoor,
		FundingRounds: len(c.FundingRounds),
	}
	for _, tag := range c.Sectors {
		p.Sectors = append(p.Sectors, tag.Value)
	}
	if c.Size != nil {
		p.Size = c.Size.Value
	}
	if c.Growth != nil {
		p.Growth = fmt.Sprintf("%+.0f%%", *c.Growth)
	}
	if c.TotalFunding != nil {
		p.FundingTotal = formatUSD(c.TotalFunding.Amount)
	}
	for _, v := range c.VisaCountries {
		if v.Location == "UK" {
			p.UKVisa = "no"
			if v.OffersVisa {
				p.UKVisa = "yes"
			}
		}
	}
	return p
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
