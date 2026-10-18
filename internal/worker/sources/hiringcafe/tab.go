package hiringcafe

import (
	"context"
	"encoding/json"
)

// Tab runs hiring.cafe requests inside a cleared hiringcafe.com page. Search returns the
// classic page's pageProps JSON; a lasting challenge returns a *sources.DeferredError.
type Tab interface {
	Search(ctx context.Context, state json.RawMessage, page int) ([]byte, error)
	Description(ctx context.Context, id string) ([]byte, error)
}
