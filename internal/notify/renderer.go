package notify

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"

	"github.com/ollymarsters/job-scraper/internal/dto"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type JobData struct {
	Title        string
	Company      string
	Location     string
	URL          string
	Remuneration string
}

type Renderer struct {
	individual *template.Template
}

func NewRenderer() (*Renderer, error) {
	individual, err := template.ParseFS(templateFS, "templates/individual.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse individual template: %w", err)
	}
	return &Renderer{individual: individual}, nil
}

func (r *Renderer) RenderIndividual(job dto.Job) (string, error) {
	var buf bytes.Buffer
	if err := r.individual.Execute(&buf, toJobData(job)); err != nil {
		return "", fmt.Errorf("render individual: %w", err)
	}
	return buf.String(), nil
}

func toJobData(j dto.Job) JobData {
	return JobData{
		Title:        j.Title,
		Company:      j.CompanySlug,
		Location:     j.Location,
		URL:          j.URL,
		Remuneration: j.SalaryRaw,
	}
}
