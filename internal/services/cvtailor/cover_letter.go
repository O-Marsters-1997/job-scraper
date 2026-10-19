package cvtailor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/docparse"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/letterwrite"
	"github.com/ollymarsters/job-scraper/internal/services/jev"
)

var errScaffoldCitation = apperr.Upstream("the model cited achievements you did not choose, try again")

func (m *Module) Scaffold(ctx context.Context, userID string, in dto.ScaffoldInput) (dto.CoverLetter, error) {
	if in.JobID == "" || in.DocID == "" || in.TabID == "" {
		return dto.CoverLetter{}, apperr.Invalid("jobId, docId and tabId are required")
	}
	if len(in.AchievementIDs) == 0 {
		return dto.CoverLetter{}, apperr.Invalid("choose at least one achievement")
	}
	if hasDuplicates(in.AchievementIDs) {
		return dto.CoverLetter{}, apperr.Invalid("achievementIds must not repeat")
	}
	isLetter, err := m.store.IsLetterTab(ctx, userID, in.DocID, in.TabID)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	if !isLetter {
		return dto.CoverLetter{}, apperr.Invalid("the source must be a tab of a letter doc you track")
	}
	achievements, err := m.chosenAchievements(ctx, userID, in.AchievementIDs)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	key, err := m.creds.Get(ctx, userID, jev.Provider)
	if errors.Is(err, data.ErrNotFound) {
		return dto.CoverLetter{}, apperr.Unprocessable("connect an OpenRouter key in Settings, AI to write a cover letter")
	}
	if err != nil {
		return dto.CoverLetter{}, fmt.Errorf("load credential: %w", err)
	}
	brief, err := m.store.JobBrief(ctx, in.JobID)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	skeleton, err := m.store.GetLetterSkeleton(ctx, userID)
	if err != nil {
		return dto.CoverLetter{}, err
	}
	paragraphs, err := m.sourceParagraphs(ctx, userID, in.DocID, in.TabID)
	if err != nil {
		return dto.CoverLetter{}, err
	}

	scaffold, cost, err := m.scaffoldCitingOnlyInput(ctx, key, letterwrite.Input{
		JobDescription: truncateRunes(brief.Description, maxExplainJobChars), Company: brief.Company, Role: brief.Title,
		Sections: skeleton.Sections, SourceParagraphs: paragraphs, Achievements: achievements,
	})
	if err != nil {
		return dto.CoverLetter{}, err
	}

	tabID, err := m.writeScaffold(ctx, userID, in.DocID, brief, scaffoldText(skeleton, paragraphs, scaffold))
	if err != nil {
		return dto.CoverLetter{}, err
	}
	return m.store.CreateCoverLetter(ctx, userID, dto.NewCoverLetter{
		JobID: in.JobID, SourceDocID: in.DocID, SourceTabID: in.TabID, ScaffoldTabID: tabID,
		AchievementIDs: in.AchievementIDs, Model: letterwrite.Model, Cost: cost,
	})
}

func (s *Service) ListCoverLetters(ctx context.Context, userID string, q dto.CoverLetterJobQuery) ([]dto.CoverLetter, error) {
	return s.store.ListJobCoverLetters(ctx, userID, q.JobID)
}

func (m *Module) chosenAchievements(ctx context.Context, userID string, ids []string) ([]letterwrite.Achievement, error) {
	positions, err := m.store.ListPositions(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]letterwrite.Achievement, len(ids))
	for i, id := range ids {
		a, ok := findAchievement(positions, id)
		if !ok {
			return nil, apperr.Invalid("unknown achievement")
		}
		out[i] = letterwrite.Achievement{ID: a.ID, Text: a.Text}
	}
	return out, nil
}

func (m *Module) sourceParagraphs(ctx context.Context, userID, docID, tabID string) ([]string, error) {
	raw, err := m.docs.GetDocument(ctx, userID, docID, tabID)
	if err != nil {
		return nil, fmt.Errorf("load letter tab: %w", err)
	}
	text, err := docparse.Text(raw)
	if err != nil {
		return nil, errors.Join(apperr.Unprocessable("could not read the letter tab"), err)
	}
	if text == "" {
		return nil, apperr.Unprocessable("the letter tab has no text")
	}
	return strings.Split(text, "\n"), nil
}

func (m *Module) scaffoldCitingOnlyInput(ctx context.Context, key string, in letterwrite.Input) (letterwrite.Scaffold, float64, error) {
	var cost float64
	for range 2 {
		res, err := m.scaffolder.Scaffold(ctx, key, in)
		cost += res.Cost
		if err != nil {
			return letterwrite.Scaffold{}, cost, suggestError(err)
		}
		if !validSourceIndexes(res.Scaffold, len(in.SourceParagraphs)) {
			return letterwrite.Scaffold{}, cost, apperr.Upstream("the model returned a scaffold that does not fit the letter, try again")
		}
		if citesOnly(res.Scaffold, in.Achievements) {
			return res.Scaffold, cost, nil
		}
	}
	return letterwrite.Scaffold{}, cost, errScaffoldCitation
}

func validSourceIndexes(s letterwrite.Scaffold, n int) bool {
	for _, p := range s.Paragraphs {
		if p.SourceIndex < 0 || p.SourceIndex >= n {
			return false
		}
	}
	return true
}

func citesOnly(s letterwrite.Scaffold, achievements []letterwrite.Achievement) bool {
	known := make(map[string]bool, len(achievements))
	for _, a := range achievements {
		known[a.ID] = true
	}
	for _, p := range s.Paragraphs {
		for _, b := range p.Bullets {
			for _, id := range b.AchievementIDs {
				if !known[id] {
					return false
				}
			}
		}
	}
	return true
}

func (m *Module) writeScaffold(ctx context.Context, userID, docID string, brief dto.JobBrief, text string) (string, error) {
	tabID, err := m.drive.AddTab(ctx, userID, docID, brief.Company+" — "+brief.Title)
	if err != nil {
		return "", fmt.Errorf("add scaffold tab: %w", err)
	}
	req, err := json.Marshal(map[string]any{"insertText": map[string]any{
		"location": map[string]any{"index": 1, "tabId": tabID},
		"text":     text,
	}})
	if err == nil {
		err = m.drive.BatchUpdate(ctx, userID, docID, []json.RawMessage{req})
	}
	if err != nil {
		return "", errors.Join(fmt.Errorf("write scaffold tab: %w", err), m.deleteTab(ctx, userID, docID, tabID))
	}
	return tabID, nil
}

func (m *Module) deleteTab(ctx context.Context, userID, docID, tabID string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	req, err := json.Marshal(map[string]any{"deleteTab": map[string]string{"tabId": tabID}})
	if err != nil {
		return fmt.Errorf("marshal deleteTab: %w", err)
	}
	if err := m.drive.BatchUpdate(ctx, userID, docID, []json.RawMessage{req}); err != nil {
		return fmt.Errorf("delete scaffold tab: %w", err)
	}
	return nil
}

func scaffoldText(sk dto.LetterSkeleton, source []string, s letterwrite.Scaffold) string {
	var lines []string
	if sk.Intro != "" {
		lines = append(lines, sk.Intro)
	}
	for _, p := range s.Paragraphs {
		lines = append(lines, source[p.SourceIndex])
		for _, b := range p.Bullets {
			prefix := "• "
			if b.Gap {
				prefix += "[GAP] "
			}
			lines = append(lines, prefix+b.Text)
		}
	}
	if sk.Conclusion != "" {
		lines = append(lines, sk.Conclusion)
	}
	return strings.Join(lines, "\n")
}
