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
	props, err := s.stamped(ctx, userID, in.SubjectID)
	if err != nil {
		return struct{}{}, err
	}
	props.Reason = in.Reason
	return struct{}{}, s.insert(ctx, nil, userID, in.Type, in.SubjectID, props)
}

func (s *Service) recordApplicationEvent(ctx context.Context, tx pgx.Tx, userID, eventType, applicationID string, p applicationProps) error {
	stamped, err := s.stamped(ctx, userID, p.JobID)
	if err != nil {
		slog.WarnContext(ctx, "application event recorded without score", slog.Any(logger.KeyErr, err))
	}
	p.jobProps = stamped
	return s.insert(ctx, tx, userID, eventType, applicationID, p)
}

func (s *Service) stamped(ctx context.Context, userID, jobID string) (jobProps, error) {
	var props jobProps
	if jobID == "" {
		return props, nil
	}
	snap, err := s.snapshots.ScoreSnapshot(ctx, userID, jobID)
	if err != nil {
		return props, fmt.Errorf("events.stamped: %w", err)
	}
	if snap != nil {
		props.Score, props.Band = &snap.Score, snap.Band
		props.ScoreModel, props.ScoreFingerprint, props.Breakdown = snap.Model, snap.Fingerprint, snap.Breakdown
	}
	return props, nil
}

func (s *Service) insert(ctx context.Context, tx pgx.Tx, userID, eventType, subjectID string, props any) error {
	raw, err := json.Marshal(props)
	if err != nil {
		return fmt.Errorf("events.insert: marshal props: %w", err)
	}
	return s.store.InsertEvent(ctx, tx, userID, eventType, subjectID, raw)
}
