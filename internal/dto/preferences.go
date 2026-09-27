package dto

// Pick is a user's stance on one bank option.
type Pick struct {
	OptionID string `json:"optionId"`
	Stance   string `json:"stance"`
	Source   string `json:"source"`
}

// Preferences is a user's editable scoring input: their picks against the
// option bank, a salary floor, and a no-go tech list.
type Preferences struct {
	Picks       []Pick   `json:"picks"`
	SalaryFloor *Money   `json:"salaryFloor"`
	BlockedTech []string `json:"blockedTech"`
}
