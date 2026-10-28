package cvtemplatestest

import (
	"testing"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtemplates"
)

func Track(t *testing.T, st cvtemplates.Store, userID, docID string) string {
	t.Helper()
	return TrackKind(t, st, userID, docID, dto.TrackedDocCV)
}

func TrackKind(t *testing.T, st cvtemplates.Store, userID, docID string, kind dto.TrackedDocKind) string {
	t.Helper()
	ctx := t.Context()
	if err := st.AddTrackedDoc(ctx, dto.AddTrackedDocInput{UserID: userID, DocID: docID, Kind: kind}); err != nil {
		t.Fatalf("AddTrackedDoc(%q, %q, %q) err = %v", userID, docID, kind, err)
	}
	docs, err := st.ListTrackedDocs(ctx, userID, kind)
	if err != nil {
		t.Fatalf("ListTrackedDocs(%q, %q) err = %v", userID, kind, err)
	}
	for _, d := range docs {
		if d.DocID == docID {
			return d.ID
		}
	}
	t.Fatalf("ListTrackedDocs(%q, %q) = %+v, want a doc with DocID %q", userID, kind, docs, docID)
	return ""
}
