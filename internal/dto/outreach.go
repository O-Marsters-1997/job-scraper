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

type DraftQuery struct {
	ContactID     string `json:"-" path:"id"`
	TemplateID    string `json:"templateId"`    // empty: the default of the Contact's suggested kind
	AchievementID string `json:"achievementId"` // empty: the matched Achievement
}

type OutreachDraft struct {
	TemplateID    string       `json:"templateId"`
	Kind          OutreachKind `json:"kind"`
	Body          string       `json:"body"`
	AchievementID string       `json:"achievementId"`
	HasHookSlot   bool         `json:"hasHookSlot"`
}

type OutreachSubject struct {
	Name           string
	Role           string
	Company        string
	JobID          string
	JobTitle       string
	SentSinceReply int
}
