package scraper

import (
	"context"
	"errors"
	"time"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/builder"
	"github.com/ollymarsters/job-scraper/internal/worker/sources/wttj"
)

const (
	MethodUserConfirmed = "user_confirmed"
	MethodWTTJOrigin    = "wttj-origin"
)

// Verification is the evidence that a Board belongs to its Company. CompanyName is the
// name the Board's source gave the company, when it gave one.
type Verification struct {
	Method      string
	CompanyName string
}

// VerifyBoard confirms the Board exists. A wttj Board needs only its company page,
// whose existence is the evidence because the token came from WTTJ's own sitemap.
func VerifyBoard(ctx context.Context, source, token string) (Verification, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if role, ok := sourcespec.SourceRole(source); !ok || role != sourcespec.RoleATS || !sourcespec.ValidBoardToken(token) {
		return Verification{}, errors.New("unsupported board configuration")
	}
	if source == "wttj" {
		name, err := wttj.CompanyName(ctx, token)
		if err != nil {
			return Verification{}, err
		}
		return Verification{Method: MethodWTTJOrigin, CompanyName: name}, nil
	}
	src, ok := builder.BuildSource(dto.SourceTarget{Source: source, Value: token, Enabled: true})
	if !ok {
		return Verification{}, errors.New("unsupported board source")
	}
	if _, _, err := src.FetchPage(ctx, ""); err != nil {
		return Verification{}, err
	}
	return Verification{Method: MethodUserConfirmed}, nil
}
