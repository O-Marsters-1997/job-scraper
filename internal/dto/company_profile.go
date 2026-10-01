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

type CompanyProfile struct {
	Sectors       []string `json:"sectors"`
	Size          string   `json:"size"`
	Growth        string   `json:"growth"`
	FundingTotal  string   `json:"funding_total"`
	FundingRounds int      `json:"funding_rounds"`
	Investors     []string `json:"investors"`
	HQ            string   `json:"hq"`
	HybridNote    string   `json:"hybrid_note"`
	UKVisa        string   `json:"uk_visa"`
	Glassdoor     string   `json:"glassdoor"`
	Mission       string   `json:"mission"`
}
