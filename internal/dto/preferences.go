package dto

// Pick is a user's stance on one bank option.
type Pick struct {
	OptionID string `json:"optionId"`
	Stance   string `json:"stance"`
	Source   string `json:"source"`
}

// CustomQuestion is a user's own atomic yes/no question, scored like a pick
// but with no bank option or dimension behind it.
type CustomQuestion struct {
	Question string `json:"question"`
	Stance   string `json:"stance"`
	Source   string `json:"source"`
}

// Preferences is a user's editable scoring input: their picks against the
// option bank, a salary floor, a no-go tech list, and any custom questions.
type Preferences struct {
	Picks       []Pick           `json:"picks"`
	SalaryFloor *Money           `json:"salaryFloor"`
	BlockedTech []string         `json:"blockedTech"`
	Customs     []CustomQuestion `json:"customs"`
}
