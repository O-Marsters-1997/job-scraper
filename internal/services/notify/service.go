package notify

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

//go:embed templates/individual.tmpl
var templateFS embed.FS

var individualTemplate = template.Must(template.ParseFS(templateFS, "templates/individual.tmpl"))

type JobData struct {
	Title        string
	Company      string
	Location     string
	URL          string
	Remuneration string
}

type NotificationService struct {
	notifier Notifier
}

func NewNotificationService(notifier Notifier) *NotificationService {
	return &NotificationService{notifier: notifier}
}

func (s *NotificationService) NotifyNewJob(ctx context.Context, job dto.Job, recipientEmail string) error {
	if recipientEmail == "" {
		return nil
	}
	var html bytes.Buffer
	data := JobData{
		Title:        job.Title,
		Company:      job.CompanySlug,
		Location:     job.Location,
		URL:          job.URL,
		Remuneration: job.SalaryRaw,
	}
	if err := individualTemplate.Execute(&html, data); err != nil {
		return fmt.Errorf("notify: render individual: %w", err)
	}
	return s.notifier.Send(ctx, recipientEmail, "New job: "+job.Title, html.String())
}
