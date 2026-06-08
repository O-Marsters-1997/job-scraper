export interface Job {
	ID: string;
	Title: string;
	Location: string;
	URL: string;
	CompanySlug: string;
	Source: string;
	UpdatedAt: string;
	ScrapedAt: string;
	DaysInOffice?: number | null;
}
