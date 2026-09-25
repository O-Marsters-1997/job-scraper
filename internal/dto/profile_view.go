package dto

type ProfileView struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UpdateProfileInput struct {
	Email string `json:"email"`
}
