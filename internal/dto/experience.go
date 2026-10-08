package dto

type Position struct {
	ID           string        `json:"id"`
	Employer     string        `json:"employer"`
	Title        string        `json:"title"`
	StartDate    *string       `json:"startDate"`
	EndDate      *string       `json:"endDate"`
	Achievements []Achievement `json:"achievements"`
}

type Achievement struct {
	ID         string `json:"id"`
	PositionID string `json:"positionId"`
	Text       string `json:"text"`
}

// PositionInput dates are YYYY-MM-DD; a nil EndDate means the Position is current.
type PositionInput struct {
	ID        string  `json:"-" path:"id"`
	Employer  string  `json:"employer"`
	Title     string  `json:"title"`
	StartDate *string `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

type AchievementInput struct {
	ID         string `json:"-" path:"id"`
	PositionID string `json:"-" path:"positionId"`
	Text       string `json:"text"`
}

type ReorderInput struct {
	PositionID string   `json:"-" path:"positionId"`
	IDs        []string `json:"ids"`
}

type BankSkill struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	SortOrder int    `json:"sortOrder"`
}

type BankSkillInput struct {
	ID       string `json:"-" path:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}
