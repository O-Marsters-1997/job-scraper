package notify

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/services/notify/store"
)

// Ntfy configures phone delivery; a zero URL disables it.
type Ntfy struct{ URL, Token string }

type Module struct {
	store Store
	phone PhonePusher
}

type Deps struct {
	Store Store
	Phone PhonePusher
}

func Build(deps Deps) *Module {
	return &Module{store: deps.Store, phone: deps.Phone}
}

func New(pool *pgxpool.Pool, ntfy Ntfy) *Module {
	deps := Deps{Store: store.New(pool)}
	if ntfy.URL != "" {
		deps.Phone = NewNtfyPusher(ntfy.URL, ntfy.Token)
	}
	return Build(deps)
}

// NewJob records a Notification for userID about job and sends it to the phone.
func (m *Module) NewJob(ctx context.Context, userID string, job dto.Job) error {
	summary, err := m.store.GetNotificationJob(ctx, userID, job.ID)
	if err != nil {
		return fmt.Errorf("get notification job: %w", err)
	}
	title, body := "New job: "+job.Title, pushBody(summary)
	id, err := m.store.InsertNotification(ctx, userID, job.ID, title, body)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	if m.phone == nil {
		return nil
	}
	msg := dto.PushMessage{Title: title, Body: body, URL: fmt.Sprintf("fasttrack://jobs/%s?n=%s", job.ID, id)}
	if err := m.phone.Send(ctx, msg); err != nil {
		slog.ErrorContext(ctx, "ntfy send failed", slog.String(logger.KeyUserID, userID), slog.Any(logger.KeyErr, err))
	}
	return nil
}

func (m *Module) NewJobByID(ctx context.Context, userID, jobID string) error {
	summary, err := m.store.GetNotificationJob(ctx, userID, jobID)
	if err != nil {
		return fmt.Errorf("get notification job: %w", err)
	}
	return m.NewJob(ctx, userID, dto.Job{ID: jobID, Title: summary.Title})
}

func pushBody(job dto.NotificationJob) string {
	var bandScore []string
	if job.Band != "" {
		bandScore = append(bandScore, strings.ToUpper(job.Band[:1])+job.Band[1:])
	}
	if job.Score != nil {
		bandScore = append(bandScore, strconv.Itoa(*job.Score))
	}
	parts := slices.DeleteFunc([]string{job.CompanyName, job.Location, strings.Join(bandScore, " ")}, func(s string) bool { return s == "" })
	return strings.Join(parts, " · ")
}
