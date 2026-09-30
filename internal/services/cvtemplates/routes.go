package cvtemplates

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/handlers"
)

func (m *Module) Routes(r chi.Router) {
	r.Route("/cv-templates", func(r chi.Router) {
		r.Get("/", handlers.GetAll(m.svc.List))
		r.Get("/{docId}/{tabId}/pdf", exportCV(m.svc))
	})

	r.Route("/tracked-docs", func(r chi.Router) {
		r.Post("/", handlers.Create(m.svc.AddDoc))
		r.Delete("/{id}", handlers.Delete(m.store.RemoveTrackedDoc))
		r.Post("/{docId}/tabs/{tabId}/hide", handlers.Update(m.svc.HideTab))
		r.Post("/{docId}/tabs/{tabId}/show", handlers.Update(m.svc.ShowTab))
	})
}

type cvExporter interface {
	ExportPDF(ctx context.Context, userID, docID, tabID string) (io.ReadCloser, error)
}

type exportRequest struct {
	userID, docID, tabID string
}

func exportCV(svc cvExporter) http.HandlerFunc {
	return handlers.Handle(
		func(r *http.Request) (exportRequest, error) {
			uid, err := handlers.UserID(r)
			if err != nil {
				return exportRequest{}, err
			}
			return exportRequest{userID: uid, docID: chi.URLParam(r, "docId"), tabID: chi.URLParam(r, "tabId")}, nil
		},
		func(ctx context.Context, in exportRequest) (io.ReadCloser, error) {
			return svc.ExportPDF(ctx, in.userID, in.docID, in.tabID)
		},
		func(w http.ResponseWriter, _ *http.Request, body io.ReadCloser) {
			defer func() { _ = body.Close() }()
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `inline; filename="cv.pdf"`)
			_, _ = io.Copy(w, body)
		},
	)
}
