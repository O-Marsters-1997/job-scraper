package hiringcafe

type pageProps struct {
	Hits       []hit `json:"ssrHits"`
	IsLastPage bool  `json:"ssrIsLastPage"`
	Error      any   `json:"ssrError"`
}

type hit struct {
	ID          string `json:"id"`
	CollapseKey string `json:"collapse_key"`
	ApplyURL    string `json:"apply_url"`
	IsExpired   bool   `json:"is_expired"`
	Info        struct {
		Title string `json:"title"`
	} `json:"job_information"`
	Processed struct {
		Location          string   `json:"formatted_workplace_location"`
		WorkplaceType     string   `json:"workplace_type"`
		PublishDate       string   `json:"estimated_publish_date"`
		SalaryTransparent bool     `json:"is_compensation_transparent"`
		Currency          string   `json:"listed_compensation_currency"`
		YearlyMin         *float64 `json:"yearly_min_compensation"`
		YearlyMax         *float64 `json:"yearly_max_compensation"`
	} `json:"v5_processed_job_data"`
	Company struct {
		Name string `json:"name"`
	} `json:"enriched_company_data"`
}

type description struct {
	Job struct {
		Info struct {
			Description string `json:"description"`
		} `json:"job_information"`
	} `json:"job"`
}
