package dto

const DefaultCVFilenamePattern = "cv-{company}"

type Profile struct {
	Username          string `json:"username"`
	Email             string `json:"email"`
	CVFilenamePattern string `json:"cvFilenamePattern"`
}

type UpdateProfileInput struct {
	Email string `json:"email"`
}

type UpdateCVFilenamePatternInput struct {
	Pattern string `json:"pattern"`
}
