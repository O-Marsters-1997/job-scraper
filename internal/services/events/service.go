package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/ollymarsters/job-scraper/internal/apperr"
	"github.com/ollymarsters/job-scraper/internal/data"
	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/logger"
)

// Event types, matching the event_type Postgres enum.
const (
	JobOpened                = "job_opened"
	JobDismissed             = "job_dismissed"
	ApplicationCreated       = "application_created"
	ApplicationStatusChanged = "application_status_changed"
	AlertOpened              = "alert_opened"
)

const maxReasonLen = 500

type Store interface {
	InsertEvent(ctx context.Context, tx pgx.Tx, userID, eventType, subjectID string, props []byte) error
	ListEvents(ctx context.Context, userID, eventType string) ([]dto.Event, error)
}

// ScoreSnapshotter returns userID's stored score for jobID, or nil when the
// job has none.
type ScoreSnapshotter interface {
	ScoreSnapshot(ctx context.Context, userID, jobID string) (*dto.JobScoreEvidence, error)
}

type Service struct {
	store     Store
	snapshots ScoreSnapshotter
}

func NewService(store Store, snapshots ScoreSnapshotter) *Service {
	return &Service{store: store, snapshots: snapshots}
}

type jobProps struct {
	Score            *int           `json:"score,omitempty"`
	Band             string         `json:"band,omitempty"`
	ScoreModel       string         `json:"score_model,omitempty"`
	ScoreFingerprint string         `json:"score_fingerprint,omitempty"`
	Breakdown        []dto.ScoreRow `json:"breakdown,omitempty"`
	Reason           string         `json:"reason,omitempty"`
}

type applicationProps struct {
	jobProps
	JobID        string `json:"job_id"`
	FromStatusID string `json:"from_status_id,omitempty"`
	ToStatusID   string `json:"to_status_id,omitempty"`
}

// Record stores a frontend-reported event for userID, stamping job events
// with the job's current score.
func (s *Service) Record(ctx context.Context, userID string, in dto.EventInput) (struct{}, error) {
	switch in.Type {
	case JobOpened, JobDismissed:
		if in.SubjectID == "" {
			return struct{}{}, apperr.Invalid("subject_id is required")
		}
	case AlertOpened:
	default:
		return struct{}{}, apperr.Invalid("unsupported event type")
	}
	if in.SubjectID != "" {
		if _, err := data.UUID(in.SubjectID); err != nil {
			return struct{}{}, apperr.Invalid("invalid subject_id")
		}
	}
	if len(in.Reason) > maxReasonLen {
		return struct{}{}, apperr.Invalid("reason too long")
	}
	var snap *dto.JobScoreEvidence
	if in.SubjectID != "" {
		var err error
		if snap, err = s.snapshots.ScoreSnapshot(ctx, userID, in.SubjectID); err != nil {
			return struct{}{}, fmt.Errorf("events.Record: snapshot: %w", err)
		}
	}
	props := stampFrom(snap, in.Type == JobDismissed)
	props.Reason = in.Reason
	return struct{}{}, s.insert(ctx, nil, userID, in.Type, in.SubjectID, props)
}

func (s *Service) recordApplicationEvent(ctx context.Context, tx pgx.Tx, userID, eventType, applicationID string, p applicationProps, snap *dto.JobScoreEvidence) error {
	p.jobProps = stampFrom(snap, true)
	return s.insert(ctx, tx, userID, eventType, applicationID, p)
}

// snapshot returns a best-effort score snapshot: a failed read is logged and
// reported as no score, so analytics never blocks the caller.
func (s *Service) snapshot(ctx context.Context, userID, jobID string) *dto.JobScoreEvidence {
	if jobID == "" {
		return nil
	}
	snap, err := s.snapshots.ScoreSnapshot(ctx, userID, jobID)
	if err != nil {
		slog.WarnContext(ctx, "score snapshot failed", slog.Any(logger.KeyErr, err))
		return nil
	}
	return snap
}

func stampFrom(snap *dto.JobScoreEvidence, withBreakdown bool) jobProps {
	var props jobProps
	if snap == nil {
		return props
	}
	props.Score, props.Band = &snap.Score, snap.Band
	props.ScoreModel, props.ScoreFingerprint = snap.Model, snap.Fingerprint
	if withBreakdown {
		props.Breakdown = snap.Breakdown
	}
	return props
}

func (s *Service) insert(ctx context.Context, tx pgx.Tx, userID, eventType, subjectID string, props any) error {
	raw, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("events.insert: marshal props: %w", err)
	}
	return s.store.InsertEvent(ctx, tx, userID, eventType, subjectID, raw)
}
