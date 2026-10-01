package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/services/cvtailor/cvedit"
)

const docURLPrefix = "https://docs.google.com/document/d/"

// GetDraft returns the Draft with, once it is ready, the provenance of its
// bullets.
func (s *Service) GetDraft(ctx context.Context, userID string, q dto.DraftQuery) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	draft = withDocURL(draft)
	if draft.Status != statusReady || len(draft.EditSet) == 0 {
		return draft, nil
	}
	positions, err := s.store.ListPositions(ctx, userID)
	if err != nil {
		return dto.Draft{}, err
	}
	var edits cvedit.EditSet
	if err := json.Unmarshal(draft.EditSet, &edits); err != nil {
		return dto.Draft{}, fmt.Errorf("decode edit set: %w", err)
	}
	draft.Provenance = provenance(edits, positions)
	content := editedContent(edits)
	draft.Content = &content
	if len(draft.BaseContent) > 0 {
		draft.Base = new(dto.DraftContent)
		if err := json.Unmarshal(draft.BaseContent, draft.Base); err != nil {
			return dto.Draft{}, fmt.Errorf("decode base content: %w", err)
		}
	}
	return draft, nil
}

// ListJobDrafts returns the User's Drafts of one Job, newest first.
func (s *Service) ListJobDrafts(ctx context.Context, userID string, q dto.DraftJobQuery) ([]dto.Draft, error) {
	drafts, err := s.store.ListJobDrafts(ctx, userID, q.JobID)
	if err != nil {
		return nil, err
	}
	for i := range drafts {
		drafts[i] = withDocURL(drafts[i])
	}
	return drafts, nil
}

// KeepDraft marks a ready Draft kept. A Job holds one kept Draft, so keeping
// a second is a conflict.
func (s *Service) KeepDraft(ctx context.Context, userID string, q dto.DraftQuery) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be kept")
	}
	switch outcome(draft) {
	case dto.OutcomeDiscarded:
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	case dto.OutcomeKept:
		return withDocURL(draft), nil
	}
	draft, err = s.store.SetDraftOutcome(ctx, userID, q.ID, dto.OutcomeKept)
	if err != nil {
		return dto.Draft{}, err
	}
	return withDocURL(draft), nil
}

// DiscardDraft deletes the Draft's Drive file and marks it discarded; the
// row stays.
func (s *Service) DiscardDraft(ctx context.Context, userID string, q dto.DraftQuery) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if draft.Status != statusReady {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be discarded")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return draft, nil
	}
	if draft.DraftDocID != "" {
		if err := s.drive.DeleteFile(ctx, userID, draft.DraftDocID); err != nil {
			return dto.Draft{}, apperr.Upstream("failed to delete the draft from Google Drive")
		}
	}
	return s.store.SetDraftOutcome(ctx, userID, q.ID, dto.OutcomeDiscarded)
}

// DraftPDF streams Google's PDF export of the Draft's Doc. The caller must
// close the body.
func (s *Service) DraftPDF(ctx context.Context, userID string, q dto.DraftQuery) (io.ReadCloser, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return nil, err
	}
	if draft.Status != statusReady || draft.DraftDocID == "" {
		return nil, apperr.NotFound("draft has no document")
	}
	body, err := s.drive.ExportPDF(ctx, userID, draft.DraftDocID, "")
	if err != nil {
		return nil, apperr.Upstream("failed to export PDF")
	}
	return body, nil
}

func outcome(d dto.Draft) string {
	if d.Outcome == nil {
		return ""
	}
	return *d.Outcome
}

func withDocURL(d dto.Draft) dto.Draft {
	if d.DraftDocID != "" && d.Status == statusReady {
		url := docURLPrefix + strings.TrimSpace(d.DraftDocID) + "/edit"
		d.DraftDocURL = &url
	}
	return d
}

func editedContent(edits cvedit.EditSet) dto.DraftContent {
	c := dto.DraftContent{Profile: edits.Profile, Skills: edits.Skills, Positions: []dto.DraftPosition{}}
	if c.Skills == nil {
		c.Skills = []string{}
	}
	for _, pe := range edits.Positions {
		dp := dto.DraftPosition{PositionID: pe.PositionID, Bullets: []dto.DraftBullet{}}
		for _, b := range pe.Bullets {
			ids := b.AchievementIDs
			if ids == nil {
				ids = []string{}
			}
			dp.Bullets = append(dp.Bullets, dto.DraftBullet{Text: b.Text, AchievementIDs: ids})
		}
		c.Positions = append(c.Positions, dp)
	}
	return c
}

func provenance(edits cvedit.EditSet, bank []dto.Position) *dto.DraftProvenance {
	positionByID := make(map[string]dto.Position, len(bank))
	achievementByID := map[string]dto.Achievement{}
	for _, p := range bank {
		positionByID[p.ID] = p
		for _, a := range p.Achievements {
			achievementByID[a.ID] = a
		}
	}
	out := &dto.DraftProvenance{Positions: []dto.ProvenancePosition{}}
	for _, pe := range edits.Positions {
		p := positionByID[pe.PositionID]
		pp := dto.ProvenancePosition{PositionID: pe.PositionID, Employer: p.Employer, Title: p.Title, Bullets: []dto.ProvenanceBullet{}}
		for _, b := range pe.Bullets {
			cited := []dto.Achievement{}
			var citedText []string
			for _, id := range b.AchievementIDs {
				if a, ok := achievementByID[id]; ok {
					cited = append(cited, a)
					citedText = append(citedText, a.Text)
				}
			}
			segments := markNovel(b.Text, citedText)
			if b.Keep {
				segments = []dto.TextSegment{{Text: b.Text}}
			}
			pp.Bullets = append(pp.Bullets, dto.ProvenanceBullet{Segments: segments, Achievements: cited})
		}
		out.Positions = append(out.Positions, pp)
	}
	return out
}

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

var stopwords = map[string]bool{
	"and": true, "the": true, "for": true, "with": true, "from": true, "that": true, "this": true,
	"into": true, "over": true, "across": true, "using": true, "through": true,
}

func markNovel(text string, sources []string) []dto.TextSegment {
	known := map[string]bool{}
	for _, src := range sources {
		for _, w := range wordRe.FindAllString(strings.ToLower(src), -1) {
			known[w] = true
		}
	}
	var out []dto.TextSegment
	last := 0
	emit := func(t string, novel bool) {
		if t == "" {
			return
		}
		if n := len(out); n > 0 && out[n-1].Novel == novel {
			out[n-1].Text += t
			return
		}
		out = append(out, dto.TextSegment{Text: t, Novel: novel})
	}
	for _, loc := range wordRe.FindAllStringIndex(text, -1) {
		emit(text[last:loc[0]], false)
		word := text[loc[0]:loc[1]]
		lower := strings.ToLower(word)
		emit(word, len([]rune(word)) > 2 && !stopwords[lower] && !known[lower])
		last = loc[1]
	}
	emit(text[last:], false)
	return out
}
