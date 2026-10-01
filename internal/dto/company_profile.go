package dto

// CompanyProfileEntry rolls one picked Option up across a Company's open
// Jobs. Known counts the Jobs whose cached Answer resolved yes or no; Known
// of zero means the profile cannot say.
type CompanyProfileEntry struct {
	Dimension Dimension `json:"dimension"`
	Label     string    `json:"label"`
	Yes       int       `json:"yes"`
	Known     int       `json:"known"`
	Total     int       `json:"total"`
}
