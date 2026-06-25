package providers

import "context"

type UserEmailProvider interface {
	GetUserEmail(ctx context.Context, userID string) (string, error)
}
