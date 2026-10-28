package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/handlers"
)

const (
	suggestWriteWindow  = 30 * time.Second
	scaffoldWriteWindow = 3 * time.Minute
)

func withWriteWindow(d time.Duration, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(d))
		h(w, r)
	}
}

func (m *Module) tailorRoutes(r chi.Router) {
	r.Route("/tailoring", func(r chi.Router) {
		r.Get("/cvs/{docId}/{tabId}/headings", handlers.Query(m.svc.Headings))
		r.Put("/cvs/{docId}/{tabId}/headings", handlers.Update(m.svc.SaveHeadings))
		r.Get("/letter-skeleton", handlers.GetAll(m.store.GetLetterSkeleton))
		r.Put("/letter-skeleton", handlers.Update(m.store.SaveLetterSkeleton))
		r.Get("/jobs/{jobId}/suggestions", handlers.Query(m.svc.Suggestions))
		r.Get("/jobs/{jobId}/skill-suggestions", handlers.Query(m.svc.SkillSuggestions))
		r.Get("/jobs/{jobId}/experience-match", handlers.Query(m.svc.ExperienceMatch))
		r.Post("/jobs/{jobId}/achievements/{achievementId}/explain", handlers.Update(m.Explain))
		r.Post("/jobs/{jobId}/cover-letters", withWriteWindow(scaffoldWriteWindow, handlers.Create(m.Scaffold)))
		r.Get("/jobs/{jobId}/cover-letters", handlers.Query(m.svc.ListCoverLetters))
		r.Get("/cover-letters/{id}/check", handlers.Query(m.CheckCoverLetter))
		r.Post("/cover-letters/{id}/done", handlers.GetByID(m.store.MarkCoverLetterDone))
		r.Post("/drafts", m.createDraft())
		r.Get("/jobs/{jobId}/drafts", handlers.Query(m.svc.ListJobDrafts))
		r.Get("/drafts/{id}", handlers.Query(m.svc.GetDraft))
		r.Get("/drafts/{id}/layout", handlers.Query(m.svc.DraftLayout))
		r.Get("/drafts/{id}/lean", handlers.Query(m.svc.DraftLean))
		r.Get("/drafts/{id}/roles", handlers.Query(m.svc.DraftRoles))
		r.Get("/drafts/{id}/insights", handlers.Query(m.svc.DraftInsights))
		r.Post("/drafts/{id}/insights", withWriteWindow(scaffoldWriteWindow, handlers.Update(m.ExplainDraft)))
		r.Get("/drafts/{id}/pdf", m.draftPDF())
		r.Post("/drafts/{id}/profile/redraft", withWriteWindow(scaffoldWriteWindow, handlers.Update(m.RedraftProfile)))
		r.Put("/drafts/{id}/slots", handlers.Update(m.svc.SaveDraftSlots))
		r.Post("/drafts/{id}/slots/{slotId}/suggest", m.suggest())
		r.Post("/drafts/{id}/keep", handlers.Update(m.svc.KeepDraft))
		r.Post("/drafts/{id}/discard", handlers.Update(m.svc.DiscardDraft))
	})
}

func (m *Module) createDraft() http.HandlerFunc {
	type req struct {
		userID string
		in     dto.DraftInput
	}
	return handlers.Handle(
		func(r *http.Request) (req, error) {
			uid, err := handlers.UserID(r)
			if err != nil {
				return req{}, err
			}
			in, err := handlers.DecodeBody[dto.DraftInput](r)
			return req{userID: uid, in: in}, err
		},
		func(ctx context.Context, in req) (dto.DraftRef, error) {
			return m.svc.CreateDraft(ctx, in.userID, in.in)
		},
		func(w http.ResponseWriter, _ *http.Request, ref dto.DraftRef) {
			handlers.WriteJSON(w, http.StatusAccepted, ref)
		},
	)
}

func (m *Module) draftPDF() http.HandlerFunc {
	type req struct {
		userID string
		q      dto.DraftQuery
	}
	return handlers.Handle(
		func(r *http.Request) (req, error) {
			uid, err := handlers.UserID(r)
			return req{userID: uid, q: dto.DraftQuery{ID: chi.URLParam(r, "id")}}, err
		},
		func(ctx context.Context, in req) (io.ReadCloser, error) {
			return m.svc.DraftPDF(ctx, in.userID, in.q)
		},
		func(w http.ResponseWriter, _ *http.Request, body io.ReadCloser) {
			defer func() { _ = body.Close() }()
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `inline; filename="cv-draft.pdf"`)
			_, _ = io.Copy(w, body)
		},
	)
}

func (m *Module) suggest() http.HandlerFunc {
	type req struct {
		userID string
		in     dto.SuggestInput
	}
	return handlers.Handle(
		func(r *http.Request) (req, error) {
			uid, err := handlers.UserID(r)
			if err != nil {
				return req{}, err
			}
			in, err := handlers.DecodeBody[dto.SuggestInput](r)
			in.ID, in.SlotID = chi.URLParam(r, "id"), chi.URLParam(r, "slotId")
			return req{userID: uid, in: in}, err
		},
		func(ctx context.Context, in req) (iter.Seq[SuggestEvent], error) {
			return m.Suggest(ctx, in.userID, in.in)
		},
		func(w http.ResponseWriter, _ *http.Request, events iter.Seq[SuggestEvent]) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			rc := http.NewResponseController(w)
			_ = rc.SetWriteDeadline(time.Now().Add(suggestWriteWindow))
			w.WriteHeader(http.StatusOK)
			_ = rc.Flush()
			for ev := range events {
				_ = rc.SetWriteDeadline(time.Now().Add(suggestWriteWindow))
				switch {
				case ev.Err != nil:
					writeEvent(w, "error", map[string]string{"message": ev.Err.Error()})
				case ev.Done != nil:
					writeEvent(w, "done", ev.Done)
				default:
					writeEvent(w, "delta", map[string]string{"text": ev.Delta})
				}
				if rc.Flush() != nil {
					return
				}
			}
		},
	)
}

func writeEvent(w http.ResponseWriter, name string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, b)
}
