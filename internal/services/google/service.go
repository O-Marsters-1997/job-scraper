package google

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/ollymarsters/job-scraper/internal/data/providers"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

const userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

type Client interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	SaveToken(ctx context.Context, userID string, tok *oauth2.Token) error
	HTTPClientForUser(ctx context.Context, userID string) (*http.Client, error)
	DeleteToken(ctx context.Context, userID string) error
}

type Service struct {
	client Client
}

func New(client Client) *Service {
	return &Service{client: client}
}

func (s *Service) AuthURL(state string) string {
	return s.client.AuthURL(state)
}

func (s *Service) Connect(ctx context.Context, userID, code string) error {
	tok, err := s.client.Exchange(ctx, code)
	if err != nil {
		return err
	}
	return s.client.SaveToken(ctx, userID, tok)
}

// Status reports whether the user has a usable Google link, fetching their
// email from the UserInfo endpoint when they do.
func (s *Service) Status(ctx context.Context, userID string) (dto.GoogleStatus, error) {
	hc, err := s.client.HTTPClientForUser(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, providers.ErrGoogleTokenNotFound):
			return dto.GoogleStatus{Connected: false}, nil
		case errors.Is(err, providers.ErrGoogleTokenUnusable):
			slog.Warn("google token unusable, treating as disconnected", slog.Any("err", err))
			return dto.GoogleStatus{Connected: false}, nil
		default:
			return dto.GoogleStatus{}, err
		}
	}

	resp, err := hc.Get(userInfoURL)
	if err != nil {
		return dto.GoogleStatus{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var info struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return dto.GoogleStatus{}, err
	}
	return dto.GoogleStatus{Connected: true, Email: info.Email}, nil
}

// Disconnect removes the user's stored Google token.
func (s *Service) Disconnect(ctx context.Context, userID, _ string) error {
	return s.client.DeleteToken(ctx, userID)
}
