export interface ScoreRow {
	key: string;
	label: string;
	stance: string;
	resolved: "yes" | "no" | "unknown";
	effect: "meets" | "misses" | "unknown" | "neutral";
	overridden: boolean;
}

export interface Job {
	ID: string;
	Title: string;
	Location: string;
	URL: string;
	CompanySlug: string;
	CompanyID?: string;
	Source: string;
	UpdatedAt: string;
	ScrapedAt: string;
	DaysInOffice?: number | null;
	WorkArrangement?: string;
	SalaryRaw?: string;
	SuitabilityScore: number | null;
	Breakdown?: ScoreRow[] | null;
	Hidden?: boolean;
	// Optional rich fields — populated in demo mode; absent from the live backend
	Description?: string;
	Skills?: string[];
	EmploymentType?: string;
	ExperienceLevel?: string;
	TeamName?: string;
	CompanySize?: string;
	SalaryRange?: string;
}
