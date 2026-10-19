package dto

import "time"

type LetterSkeleton struct {
	Intro      string   `json:"intro"`
	Conclusion string   `json:"conclusion"`
	Sections   []string `json:"sections"`
}

type ScaffoldInput struct {
	JobID          string   `json:"-" path:"jobId"`
	DocID          string   `json:"docId"`
	TabID          string   `json:"tabId"`
	AchievementIDs []string `json:"achievementIds"`
}

type CoverLetterJobQuery struct {
	JobID string `json:"-" path:"jobId"`
}

type CoverLetter struct {
	ID             string    `json:"id"`
	JobID          string    `json:"jobId"`
	SourceDocID    string    `json:"sourceDocId"`
	SourceTabID    string    `json:"sourceTabId"`
	TabURL         string    `json:"tabUrl"`
	AchievementIDs []string  `json:"achievementIds"`
	Cost           float64   `json:"cost"`
	CreatedAt      time.Time `json:"createdAt"`
}

type NewCoverLetter struct {
	JobID          string
	SourceDocID    string
	SourceTabID    string
	ScaffoldTabID  string
	AchievementIDs []string
	Model          string
	Cost           float64
}

type JobBrief struct {
	Title       string
	Company     string
	Description string
}
