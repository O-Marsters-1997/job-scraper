package identity

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/google"
)

const userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

func (s *Service) AuthURL(state string, write bool) string {
	return s.google.AuthURL(state, write)
}

func (s *Service) Connect(ctx context.Context, userID, code string) error {
	tok, err := s.google.Exchange(ctx, code)
	if err != nil {
		return err
	}
	return s.google.SaveToken(ctx, userID, tok)
}

func (s *Service) GoogleStatus(ctx context.Context, userID string) (dto.GoogleStatus, error) {
	hc, err := s.google.HTTPClientForUser(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, google.ErrTokenNotFound):
			return dto.GoogleStatus{Connected: false}, nil
		case errors.Is(err, google.ErrTokenUnusable):
			slog.WarnContext(ctx, "google token unusable, treating as disconnected", slog.Any(logger.KeyErr, err))
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
	canWrite, err := s.google.HasScope(ctx, userID, google.DriveFileScope)
	if err != nil {
		return dto.GoogleStatus{}, err
	}
	canEditDocs, err := s.google.HasScope(ctx, userID, google.DocumentsScope)
	if err != nil {
		return dto.GoogleStatus{}, err
	}
	return dto.GoogleStatus{Connected: true, Email: info.Email, CanWrite: canWrite, CanEditDocs: canEditDocs}, nil
}

func (s *Service) DisconnectGoogle(ctx context.Context, userID, _ string) error {
	return s.google.DeleteToken(ctx, userID)
}
