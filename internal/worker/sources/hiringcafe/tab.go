package hiringcafe

import (
	"context"
	"encoding/json"
)

type Tab interface {
	Search(ctx context.Context, state json.RawMessage, page int) ([]byte, error)
	Description(ctx context.Context, id string) ([]byte, error)
}
