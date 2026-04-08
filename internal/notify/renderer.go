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

// JobData is the template-friendly representation of a job.
type JobData struct {
	Title        string
	Company      string
	Location     string
	URL          string
	Remuneration string
}

type digestTemplateData struct {
	Jobs []JobData
}

// Renderer renders email templates to HTML strings.
type Renderer struct {
	digest     *template.Template
	individual *template.Template
}

// NewRenderer parses and returns a Renderer backed by the embedded templates.
func NewRenderer() (*Renderer, error) {
	digest, err := template.ParseFS(templateFS, "templates/digest.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse digest template: %w", err)
	}
	individual, err := template.ParseFS(templateFS, "templates/individual.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse individual template: %w", err)
	}
	return &Renderer{digest: digest, individual: individual}, nil
}

// RenderDigest renders the digest template with the given jobs.
func (r *Renderer) RenderDigest(jobs []dto.Job) (string, error) {
	data := digestTemplateData{Jobs: toJobDataSlice(jobs)}
	var buf bytes.Buffer
	if err := r.digest.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render digest: %w", err)
	}
	return buf.String(), nil
}

// RenderIndividual renders the individual job template.
func (r *Renderer) RenderIndividual(job dto.Job) (string, error) {
	var buf bytes.Buffer
	if err := r.individual.Execute(&buf, toJobData(job)); err != nil {
		return "", fmt.Errorf("render individual: %w", err)
	}
	return buf.String(), nil
}

func toJobData(j dto.Job) JobData {
	return JobData{
		Title:    j.Title,
		Company:  j.CompanySlug,
		Location: j.Location,
		URL:      j.URL,
	}
}

func toJobDataSlice(jobs []dto.Job) []JobData {
	out := make([]JobData, len(jobs))
	for i, j := range jobs {
		out[i] = toJobData(j)
	}
	return out
}
