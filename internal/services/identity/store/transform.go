package store

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/identity/store/sqlc"
)

func toUserDTO(u sqlc.User) dto.User {
	return dto.User{
		ID:           u.ID.String(),
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Email:        u.Email.String,
	}
}

func toSessionDTO(s sqlc.Session) dto.Session {
	return dto.Session{
		ID:        s.ID.String(),
		UserID:    s.UserID.String(),
		ExpiresAt: s.ExpiresAt.Time,
	}
}

func toSessionRowDTO(r sqlc.GetSessionRow) dto.Session {
	return dto.Session{
		ID:        r.ID.String(),
		UserID:    r.UserID.String(),
		Username:  r.Username,
		Role:      string(r.Role),
		ExpiresAt: r.ExpiresAt.Time,
	}
}

func toProfileDTO(username string, email pgtype.Text) dto.Profile {
	return dto.Profile{Username: username, Email: email.String}
}

func toGoogleTokenDTO(r sqlc.GetGoogleOAuthTokenRow) dto.GoogleToken {
	return dto.GoogleToken{
		AccessTokenEnc:  r.AccessTokenEnc,
		RefreshTokenEnc: r.RefreshTokenEnc,
		TokenType:       r.TokenType,
		Expiry:          r.Expiry.Time,
		Scope:           r.Scope,
	}
}
