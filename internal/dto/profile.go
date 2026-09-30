package dto

type Profile struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UpdateProfileInput struct {
	Email string `json:"email"`
}
