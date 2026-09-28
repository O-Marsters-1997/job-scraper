// Package builder is separate from sources because sources sub-packages
// (e.g. sources/greenhouse) import sources, which would create a cycle if
// BuildSource were defined there.
package builder

import (
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/ashby"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/greenhouse"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/indeed"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/lever"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/linkedin"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/personio"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/recruitee"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/remoteok"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/remotive"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wis"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/workable"
)

// BuildSource builds the Source for one enabled target. It returns false when
// the target is disabled or its source is unknown.
func BuildSource(t dto.SourceTarget) (sources.Source, bool) {
	if !t.Enabled {
		return nil, false
	}
	switch t.Source {
	case "greenhouse":
		return greenhouse.New(t.Value), true
	case "lever":
		return lever.New(t.Value), true
	case "ashby":
		return ashby.New(t.Value), true
	case "workable":
		return workable.New(t.Value), true
	case "recruitee":
		return recruitee.New(t.Value), true
	case "personio":
		return personio.New(t.Value), true
	case "indeed":
		return indeed.New(t.Value), true
	case "remoteok":
		return remoteok.New(t.Value), true
	case "remotive":
		return remotive.New(t.Value), true
	case "wis":
		return wis.New(wis.Search{Keywords: t.Value, Region: t.Filters["region"]}), true
	case "linkedin":
		return linkedin.New(linkedin.Search{
			Keywords:    t.Value,
			Location:    t.Filters["location"],
			CompanyID:   t.Filters["company_id"],
			Recency:     t.Filters["recency"],
			Arrangement: t.Filters["arrangement"],
			Experience:  t.Filters["experience"],
			JobType:     t.Filters["job_type"],
			GeoID:       t.Filters["geo_id"],
			Distance:    t.Filters["distance"],
			SalaryBand:  t.Filters["salary_band"],
		}), true
	default:
		return nil, false
	}
}
