package dto

// Pick is a user's stance on one bank option. Overridden is computed for
// display only: true for a "text" pick whose option also has a "manual" pick,
// which always wins.
type Pick struct {
	OptionID   string `json:"optionId"`
	Stance     string `json:"stance"`
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
