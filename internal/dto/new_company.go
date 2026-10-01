package dto

type NewCompany struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Slug            string                `json:"slug"`
	Boards          []TrackedBoard        `json:"boards"`
	MatchingRoles   int                   `json:"matching_roles"`
	BestSuitability *int                  `json:"best_suitability"`
	Profile         *CompanyProfile       `json:"profile"`
	Rollup          []CompanyProfileEntry `json:"rollup"`
}
