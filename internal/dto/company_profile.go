package dto

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
