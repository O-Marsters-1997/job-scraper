package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
)

var ashbyTitle = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// DiscoverBoard fetches a Board's Jobs and its display name, falling back to
// token when the ATS exposes no usable name.
func DiscoverBoard(ctx context.Context, source, token string) (string, []dto.Job, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if role, ok := sourcespec.SourceRole(source); !ok || role != sourcespec.RoleATS || !sourcespec.ValidBoardToken(token) {
		return "", nil, errors.New("unsupported board configuration")
	}
	src, ok := builder.BuildSource(dto.SourceTarget{Source: source, Value: token, Enabled: true})
	if !ok {
		return "", nil, errors.New("unsupported board source")
	}
	jobs, _, err := src.FetchPage(ctx, "")
	if err != nil {
		return "", nil, err
	}
	base := sources.NewBase(sources.Config{Name: source})
	name := ""
	switch source {
	case "greenhouse":
		name = greenhouseName(ctx, &base, token)
	case "ashby":
		name = ashbyName(ctx, &base, token)
	}
	if name == "" {
		name = token
	}
	return name, jobs, nil
}

func greenhouseName(ctx context.Context, base *sources.PaginatedBase, token string) string {
	body, err := base.Get(ctx, fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s", token))
	if err != nil {
		return ""
	}
	var resp struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return ""
	}
	return strings.TrimSpace(resp.Name)
}

func ashbyName(ctx context.Context, base *sources.PaginatedBase, token string) string {
	body, err := base.Get(ctx, fmt.Sprintf("https://jobs.ashbyhq.com/%s", token))
	if err != nil {
		return ""
	}
	m := ashbyTitle.FindSubmatch(body)
	if m == nil {
		return ""
	}
	name := strings.TrimSpace(html.UnescapeString(string(m[1])))
	for _, suffix := range []string{" Careers", " Jobs"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}
