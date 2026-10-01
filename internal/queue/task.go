package queue

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

type TaskKind string

const (
	ListingPageTask   TaskKind = "listing_page"
	DetailTask        TaskKind = "detail"
	BoardCheckTask    TaskKind = "board_check"
	BoardVerifyTask   TaskKind = "board_verify"
	BoardDiscoverTask TaskKind = "board_discover"
)

type Task struct {
	Version     int      `json:"version"`
	ID          string   `json:"task_id"`
	Source      string   `json:"source"`
	Kind        TaskKind `json:"kind"`
	TargetID    string   `json:"target_id,omitempty"`
	RunID       string   `json:"run_id,omitempty"`
	Cursor      string   `json:"page_cursor,omitempty"`
	Page        int      `json:"page,omitempty"`
	URL         string   `json:"url,omitempty"`
	Card        dto.Job  `json:"card,omitempty"`
	BoardID     string   `json:"board_id,omitempty"`
	CompanyID   string   `json:"company_id,omitempty"`
	BoardToken  string   `json:"board_token,omitempty"`
	Manual      bool     `json:"manual,omitempty"`
	Recovery    bool     `json:"recovery,omitempty"`
	Redelivered bool     `json:"-"`
}

func (t Task) Validate() error {
	if t.Version != 1 || uuid.Validate(t.ID) != nil {
		return errors.New("unsupported task version or invalid task ID")
	}
	role, ok := sourcespec.SourceRole(t.Source)
	if !ok {
		return fmt.Errorf("unknown source %q", t.Source)
	}
	if (t.TargetID == "") != (t.RunID == "") || t.TargetID != "" && (uuid.Validate(t.TargetID) != nil || uuid.Validate(t.RunID) != nil) {
		return errors.New("invalid target/run ID pair")
	}
	switch t.Kind {
	case DetailTask:
		u, err := url.Parse(t.URL)
		if err != nil || u.Host == "" || u.User != nil || u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("invalid detail URL")
		}
		if t.Card.Source != t.Source {
			return errors.New("detail card source differs from task source")
		}
	case ListingPageTask:
		if role != sourcespec.RoleDiscovery || t.TargetID == "" {
			return errors.New("listing page requires discovery source and target/run")
		}
	case BoardCheckTask:
		if role != sourcespec.RoleATS || uuid.Validate(t.BoardID) != nil {
			return errors.New("board check requires ATS source and board ID")
		}
	case BoardVerifyTask:
		if role != sourcespec.RoleATS || uuid.Validate(t.CompanyID) != nil || !sourcespec.ValidBoardToken(t.BoardToken) {
			return errors.New("board verify requires ATS source, company ID and board token")
		}
	case BoardDiscoverTask:
		if role != sourcespec.RoleATS || !sourcespec.ValidBoardToken(t.BoardToken) {
			return errors.New("board discover requires ATS source and board token")
		}
	default:
		return fmt.Errorf("unknown task kind %q", t.Kind)
	}
	return nil
}
