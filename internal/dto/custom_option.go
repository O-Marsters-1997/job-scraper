package dto

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
