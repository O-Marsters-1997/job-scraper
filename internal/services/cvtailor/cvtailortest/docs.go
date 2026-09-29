package cvtailortest

import (
	"context"
	"encoding/json"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor"
)

type Docs struct {
	TabJSON json.RawMessage
	Err     error
}

func (d Docs) GetDocument(context.Context, string, string, string) (json.RawMessage, error) {
	return d.TabJSON, d.Err
}

var _ cvtailor.DocFetcher = Docs{}
