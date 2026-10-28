package dto

import "encoding/json"

type DraftRoles struct {
	DraftID string      `json:"draftId"`
	Kept    bool        `json:"kept"`
	Spills  bool        `json:"spills"`
	Roles   []DraftRole `json:"roles"`
}

type DraftRole struct {
	PositionID *string      `json:"positionId"`
	Employer   string       `json:"employer"`
	Title      string       `json:"title"`
	Budget     int          `json:"budget"`
	Bullets    []RoleBullet `json:"bullets"`
	Bank       []RoleBullet `json:"bank"`
	Excluded   []RoleBullet `json:"excluded"`
}

type RoleBullet struct {
	SlotID        string          `json:"slotId"`
	AchievementID *string         `json:"achievementId"`
	Text          string          `json:"text"`
	Lines         int             `json:"lines"`
	Lean          *float64        `json:"lean"`
	State         SuggestionState `json:"state"`
	OnSource      bool            `json:"onSource"`
	ReplacedFrom  *string         `json:"replacedFrom"`
}

func (b RoleBullet) MarshalJSON() ([]byte, error) {
	var state *SuggestionState
	if b.State != "" {
		state = &b.State
	}
	return json.Marshal(struct {
		SlotID        string           `json:"slotId"`
		AchievementID *string          `json:"achievementId"`
		Text          string           `json:"text"`
		Lines         int              `json:"lines"`
		Lean          *float64         `json:"lean"`
		State         *SuggestionState `json:"state"`
		OnSource      bool             `json:"onSource"`
		ReplacedFrom  *string          `json:"replacedFrom"`
	}{b.SlotID, b.AchievementID, b.Text, b.Lines, b.Lean, state, b.OnSource, b.ReplacedFrom})
}
