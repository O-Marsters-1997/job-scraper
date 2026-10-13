package identity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/services/identity/store"
	"github.com/ollymarsters/job-scraper/internal/tokencrypt"
)

func ReencryptTokens(ctx context.Context, pool *pgxpool.Pool, credCipher, googleCipher *tokencrypt.Cipher) (credentials, googleTokens int, err error) {
	st := store.New(pool)
	credentials, err = st.ReencryptAICredentials(ctx, func(userID, provider, enc string) (string, error) {
		return reseal(credCipher, enc, credentialAAD(userID, provider))
	})
	if err != nil {
		return credentials, 0, fmt.Errorf("reencrypt ai credentials: %w", err)
	}
	googleTokens, err = st.ReencryptGoogleTokens(ctx, func(userID, enc string) (string, error) {
		return reseal(googleCipher, enc, tokencrypt.AAD{Table: tokencrypt.TableGoogleOAuthTokens, Owner: userID})
	})
	if err != nil {
		return credentials, googleTokens, fmt.Errorf("reencrypt google tokens: %w", err)
	}
	return credentials, googleTokens, nil
}

func reseal(c *tokencrypt.Cipher, enc string, aad tokencrypt.AAD) (string, error) {
	if !tokencrypt.IsLegacy(enc) {
		return enc, nil
	}
	plain, err := c.Decrypt(enc, aad)
	if err != nil {
		return "", err
	}
	return c.Encrypt(plain, aad)
}
