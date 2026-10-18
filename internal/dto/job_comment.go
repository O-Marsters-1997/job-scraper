package dto

import "time"

type CommentQuote struct {
	Exact  string `json:"exact"`
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

type JobComment struct {
	ID        string
	JobID     string
	Body      string
	Quote     CommentQuote
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateJobCommentInput struct {
	JobID string       `json:"-" path:"id"`
	Body  string       `json:"body"`
	Quote CommentQuote `json:"quote"`
}

type UpdateJobCommentInput struct {
	ID   string `json:"-" path:"id"`
	Body string `json:"body"`
}
