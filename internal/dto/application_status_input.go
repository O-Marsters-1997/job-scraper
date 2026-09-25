package dto

type ApplicationStatusInput struct {
	ID     string `json:"-" path:"id"`
	Name   string `json:"name"`
	Colour string `json:"colour"`
}
