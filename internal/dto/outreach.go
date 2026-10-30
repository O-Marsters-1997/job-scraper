package dto

type OutreachTemplate struct {
	ID        string       `json:"id"`
	Kind      OutreachKind `json:"kind"`
	Name      string       `json:"name"`
	Body      string       `json:"body"`
	IsDefault bool         `json:"isDefault"`
}

type OutreachTemplateInput struct {
	ID   string       `json:"-" path:"id"`
	Kind OutreachKind `json:"kind"`
	Name string       `json:"name"`
	Body string       `json:"body"`
}

type OutreachTemplateQuery struct {
	ID string `json:"-" path:"id"`
}
