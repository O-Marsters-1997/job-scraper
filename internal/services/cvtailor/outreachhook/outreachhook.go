package outreachhook

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/lineedit"
)

const (
	Model     = "anthropic/claude-haiku-4.5"
	maxTokens = 300
)

//go:embed hook.md
var system string

type Input struct {
	JobTitle       string
	JobDescription string // truncated
	Company        string
	ContactRole    string
	Achievement    string
}

type Result struct {
	Text string
	Cost float64
}

type Writer struct{ model lineedit.Model }

func New(model lineedit.Model) *Writer { return &Writer{model: model} }

func (w *Writer) Write(ctx context.Context, apiKey string, in Input) (Result, error) {
	p := lineedit.Prompt{Model: Model, System: system, User: userPrompt(in), MaxTokens: maxTokens}
	c, err := w.model.Complete(ctx, apiKey, p, func(string) {})
	if err != nil {
		return Result{Cost: c.Cost}, fmt.Errorf("write hook: %w", err)
	}
	text := strings.Join(strings.Fields(strings.Trim(strings.TrimSpace(c.Text), `"`)), " ")
	return Result{Text: text, Cost: c.Cost}, nil
}

func userPrompt(in Input) string {
	return fmt.Sprintf("Job title: %s\nCompany: %s\nRecipient's role: %s\n\nJob description:\n%s\n\nAchievement the sender mentions next:\n%s\n",
		in.JobTitle, in.Company, in.ContactRole, in.JobDescription, in.Achievement)
}
