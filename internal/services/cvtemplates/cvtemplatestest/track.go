package cvtemplatestest

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

func Track(t *testing.T, st cvtemplates.Store, userID, docID string) string {
	t.Helper()
	ctx := t.Context()
	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID}); err != nil {
		t.Fatalf("AddTrackedDoc(%q, %q) err = %v", userID, docID, err)
	}
	docs, err := st.ListTrackedDocs(ctx, userID)
	if err != nil {
		t.Fatalf("ListTrackedDocs(%q) err = %v", userID, err)
	}
	for _, d := range docs {
		if d.DocID == docID {
			return d.ID
		}
	}
	t.Fatalf("ListTrackedDocs(%q) = %+v, want a doc with DocID %q", userID, docs, docID)
	return ""
}
