package cvtailor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
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
	draft.SkillsEditable = !edits.LegacySkills && len(content.SkillGroups) > 0
	if len(draft.BaseContent) == 0 {
		return draft, nil
	}
	var stored baseContent
	if err := json.Unmarshal(draft.BaseContent, &stored); err != nil {
		return dto.Draft{}, fmt.Errorf("decode base content: %w", err)
	}
	base := stored.content()
	draft.Base = &base
	if content.Profile == nil {
		content.Profile = draft.Base.Profile
	}
	if len(content.Skills) == 0 {
		content.Skills, content.SkillGroups = draft.Base.Skills, draft.Base.SkillGroups
		draft.SkillsEditable = !edits.LegacySkills && len(content.SkillGroups) > 0
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

// KeepDraft queues a ready Draft to be kept as a named Doc on the next tick.
// A Job holds one kept Draft, so keeping a second is a conflict.
func (s *Service) KeepDraft(ctx context.Context, userID string, q dto.DraftQuery) (dto.Draft, error) {
	draft, err := s.store.GetDraft(ctx, userID, q.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	if holdsKeep(draft) {
		return withDocURL(draft), nil
	}
	if draft.Status != statusReady {
		return dto.Draft{}, apperr.Conflict("only a ready draft can be kept")
	}
	if outcome(draft) == dto.OutcomeDiscarded {
		return dto.Draft{}, apperr.Conflict("this draft was discarded")
	}
	siblings, err := s.store.ListJobDrafts(ctx, userID, draft.JobID)
	if err != nil {
		return dto.Draft{}, err
	}
	for _, other := range siblings {
		if other.ID != draft.ID && holdsKeep(other) {
			return dto.Draft{}, apperr.Conflict("this job already has a kept draft")
		}
	}
	draft, err = s.store.QueueKeep(ctx, userID, q.ID)
	if err != nil {
		return dto.Draft{}, err
	}
	return withDocURL(draft), nil
}

func holdsKeep(d dto.Draft) bool {
	return d.Status == statusKeeping || outcome(d) == dto.OutcomeKept
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
			slog.ErrorContext(ctx, "delete discarded draft failed", slog.String("draft_id", draft.ID), slog.Any(logger.KeyErr, err))
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
	if (draft.Status != statusReady && draft.Status != statusKeeping) || draft.DraftDocID == "" {
		return nil, apperr.NotFound("draft has no document")
	}
	body, err := s.drive.ExportPDF(ctx, userID, draft.DraftDocID, "")
	if err != nil {
		slog.ErrorContext(ctx, "export draft pdf failed", slog.String("draft_id", draft.ID), slog.Any(logger.KeyErr, err))
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

func skillContent(groups []cvedit.SkillGroup) ([]string, []dto.SkillGroup) {
	flat := cvedit.FlatSkills(groups)
	if flat == nil {
		flat = []string{}
	}
	out := make([]dto.SkillGroup, len(groups))
	for i, g := range groups {
		items := g.Items
		if items == nil {
			items = []string{}
		}
		out[i] = dto.SkillGroup{Label: g.Label, Items: items}
	}
	return flat, out
}

func editedContent(edits cvedit.EditSet) dto.DraftContent {
	c := dto.DraftContent{Profile: edits.Profile, Positions: []dto.DraftPosition{}}
	c.Skills, c.SkillGroups = skillContent(edits.Skills)
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
	if edits.Profile != nil {
		out.Profile = &dto.ProvenanceProfile{SlotID: profileSlotID}
	}
	for _, pe := range edits.Positions {
		p := positionByID[pe.PositionID]
		pp := dto.ProvenancePosition{PositionID: pe.PositionID, Employer: p.Employer, Title: p.Title, Bullets: []dto.ProvenanceBullet{}}
		for _, b := range pe.Bullets {
			cited := []dto.Achievement{}
			for _, id := range b.AchievementIDs {
				if a, ok := achievementByID[id]; ok {
					cited = append(cited, a)
				}
			}
			pp.Bullets = append(pp.Bullets, dto.ProvenanceBullet{SlotID: b.SlotID, Achievements: cited})
		}
		out.Positions = append(out.Positions, pp)
	}
	return out
}
