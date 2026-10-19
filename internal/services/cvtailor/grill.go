package cvtailor

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
)

//go:embed grill.md
var grillWrapper string

var jsonBlock = regexp.MustCompile("(?s)```json\\s*(.*?)```")

type grillExperience struct {
	Positions []dto.Position  `json:"positions"`
	Skills    []dto.BankSkill `json:"skills"`
}

type grillOutput struct {
	Achievements []struct {
		ID      string `json:"id"`
		Context string `json:"context"`
	} `json:"achievements"`
}

func (s *Service) GrillPrompt(ctx context.Context, userID string, q dto.GrillPromptQuery) (dto.GrillPrompt, error) {
	if (q.DocID == "") != (q.TabID == "") {
		return dto.GrillPrompt{}, apperr.Invalid("docId and tabId go together")
	}
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.GrillPrompt{}, err
	}
	skills, err := s.store.ListBankSkills(ctx, userID)
	if err != nil {
		return dto.GrillPrompt{}, err
	}
	experience, err := json.MarshalIndent(grillExperience{Positions: positions, Skills: skills}, "", "  ")
	if err != nil {
		return dto.GrillPrompt{}, fmt.Errorf("marshal experience: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(grillWrapper))
	sb.WriteString("\n\n## My experience\n\n```json\n")
	sb.Write(experience)
	sb.WriteString("\n```\n")
	if q.DocID != "" {
		raw, err := s.docs.GetDocument(ctx, userID, q.DocID, q.TabID)
		if err != nil {
			return dto.GrillPrompt{}, fmt.Errorf("load CV tab: %w", err)
		}
		text, err := docparse.Text(raw)
		if err != nil {
			return dto.GrillPrompt{}, apperr.Unprocessable("could not read the CV tab")
		}
		sb.WriteString("\n## My CV\n\n")
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return dto.GrillPrompt{Text: sb.String()}, nil
}

func (s *Service) PreviewContext(ctx context.Context, userID string, in dto.ContextImportInput) ([]dto.ContextChange, error) {
	matches := jsonBlock.FindAllStringSubmatch(in.Output, -1)
	if len(matches) == 0 {
		return nil, apperr.Invalid("no json block found in the output")
	}
	var out grillOutput
	if err := json.Unmarshal([]byte(matches[len(matches)-1][1]), &out); err != nil {
		return nil, apperr.Invalid("the last json block is not valid")
	}
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	owned := map[string]dto.Achievement{}
	for _, p := range positions {
		for _, a := range p.Achievements {
			owned[a.ID] = a
		}
	}
	seen := make(map[string]struct{}, len(out.Achievements))
	changes := make([]dto.ContextChange, 0, len(out.Achievements))
	for _, c := range out.Achievements {
		a, ok := owned[c.ID]
		if !ok {
			return nil, apperr.NotFound("achievement " + c.ID + " not found")
		}
		if _, dup := seen[c.ID]; dup {
			return nil, apperr.Invalid("achievement " + c.ID + " appears twice")
		}
		seen[c.ID] = struct{}{}
		changes = append(changes, dto.ContextChange{
			AchievementID: a.ID, Text: a.Text, Before: a.Context, After: strings.TrimSpace(c.Context),
		})
	}
	return changes, nil
}

func (s *Service) ApplyContext(ctx context.Context, userID string, in dto.ContextApplyInput) (struct{}, error) {
	if len(in.Changes) == 0 {
		return struct{}{}, apperr.Invalid("changes are required")
	}
	ids := make([]string, len(in.Changes))
	contexts := make([]string, len(in.Changes))
	for i, c := range in.Changes {
		ids[i], contexts[i] = c.AchievementID, strings.TrimSpace(c.After)
	}
	if hasDuplicates(ids) {
		return struct{}{}, apperr.Invalid("changes must not repeat an achievement")
	}
	return struct{}{}, s.store.SetAchievementContexts(ctx, userID, ids, contexts)
}
