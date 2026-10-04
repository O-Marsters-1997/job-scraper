package dto

// Pick is a user's stance on one bank option; Weight (1-100) is set only on a
// ladder dimension's Picks. Overridden is display-only: a "text" pick whose
// option also has a "manual" pick, which always wins.
type Pick struct {
	OptionID   string `json:"optionId"`
	Stance     string `json:"stance"`
	Weight     int    `json:"weight,omitempty"`
	Source     string `json:"source"`
	Overridden bool   `json:"overridden"`
}

// Preferences is a user's editable scoring input: their picks, a salary
// floor, and the free text extraction derives text picks from.
// PreferenceTextHash is PreferenceText's hash as of the last extraction.
type Preferences struct {
	Picks              []Pick `json:"picks"`
	SalaryFloor        *Money `json:"salaryFloor"`
	PreferenceText     string `json:"preferenceText"`
	PreferenceTextHash string `json:"preferenceTextHash"`
}
