package dto

type SourceTarget struct {
	ID      string
	UserID  string
	Source  string
	Value   string
	Enabled bool
	Filters map[string]string
}
