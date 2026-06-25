package providers

import "context"

// UserAICredentialsProvider is the persistence interface for encrypted AI
// provider keys. Callers never see SQL or pgtype.
type UserAICredentialsProvider interface {
	UpsertUserAICredential(ctx context.Context, userID, provider, encKey string) error
	GetUserAICredential(ctx context.Context, userID, provider string) (string, error)
	DeleteUserAICredential(ctx context.Context, userID, provider string) error
	ListUserAICredentialProviders(ctx context.Context, userID string) ([]string, error)
}
