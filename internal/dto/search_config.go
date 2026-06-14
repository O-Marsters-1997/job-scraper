package dto

type SearchConfig struct {
	ID                string
	UserID            string
	Role              string
	Location          string
	Keywords          []string
	SuitabilityRubric string
	RelevanceCutoff   int
	NotifyThreshold   int
}
