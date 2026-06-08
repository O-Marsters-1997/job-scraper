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
	// Optional rich fields — populated in demo mode; absent from the live backend
	Description?: string;
	Skills?: string[];
	EmploymentType?: string;
	ExperienceLevel?: string;
	TeamName?: string;
	CompanySize?: string;
	SalaryRange?: string;
}
