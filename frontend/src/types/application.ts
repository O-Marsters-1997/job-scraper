export interface Application {
	ID: string;
	UserID: string;
	JobID: string;
	StatusID: string;
	Notes: string;
	AppliedAt: string | null;
	SalaryInfo: string;
	CreatedAt: string;
	UpdatedAt: string;
}

export interface ApplicationWithDetails {
	ID: string;
	UserID: string;
	JobID: string;
	JobTitle: string;
	JobCompanySlug: string;
	JobLocation: string;
	JobURL: string;
	StatusID: string;
	StatusName: string;
	StatusColour: string;
	Notes: string;
	AppliedAt: string | null;
	SalaryInfo: string;
	CreatedAt: string;
	UpdatedAt: string;
}

export interface JobApplicationSummary {
	ApplicationID: string;
	StatusID: string;
	StatusName: string;
	StatusColour: string;
}
