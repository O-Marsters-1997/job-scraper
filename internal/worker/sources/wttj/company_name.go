package wttj

import (
	"context"
	"net/url"

	"github.com/ollymarsters/job-scraper/internal/worker/sources"
)

// CompanyName fetches the company page for token and returns the company's display name.
func CompanyName(ctx context.Context, token string) (string, error) {
	base := sources.NewBase(sources.Config{Name: name})
	body, err := base.Get(ctx, baseURL+"/companies/"+url.PathEscape(token))
	if err != nil {
		return "", err
	}
	c, err := parseCompanyState(body)
	if err != nil {
		return "", err
	}
	return c.Name, nil
}
