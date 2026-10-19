package dto

type LetterSkeleton struct {
	Intro      string   `json:"intro"`
	Conclusion string   `json:"conclusion"`
	Sections   []string `json:"sections"`
}
