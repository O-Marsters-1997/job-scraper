package dto

type CustomOptionDraftInput struct {
	Phrase    string    `json:"phrase"`
	Question  string    `json:"question,omitempty"`
	Dimension Dimension `json:"dimension,omitempty"`
}

type CustomOptionDraft struct {
	Question  string             `json:"question"`
	Dimension Dimension          `json:"dimension"`
	Answers   []DraftAnswerOnJob `json:"answers"`
}

type DraftAnswerOnJob struct {
	JobID    string `json:"jobId"`
	Title    string `json:"title"`
	Company  string `json:"company"`
	Resolved string `json:"resolved"`
}

type DraftedQuestion struct {
	Question  string
	Dimension Dimension
}

type CustomOption struct {
	ID        string    `json:"id"`
	Dimension Dimension `json:"dimension"`
	Label     string    `json:"label"`
	Question  string    `json:"question"`
}

type CustomOptionInput struct {
	ID        string    `json:"-" path:"id"`
	Label     string    `json:"label"`
	Question  string    `json:"question"`
	Dimension Dimension `json:"dimension"`
}
