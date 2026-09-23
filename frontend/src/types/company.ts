export interface Company {
	ID: string;
	Slug: string;
	Name: string;
	ATSSource: string;
	ATSToken: string;
	FirstSeenAt: string;
	JobCount: number;
	Tracked: boolean;
	TargetID: string;
	CheckIntervalMinutes: number;
	LastCheckedAt: string | null;
}

export interface CompanyTracking {
	CompanyID: string;
	UserID: string;
	Enabled: boolean;
	CheckIntervalMinutes: number;
}

export interface CompanyBoard {
	ID: string;
	CompanyID: string;
	Source: string;
	BoardToken: string;
	Status: "candidate" | "verified" | "retired";
	VerificationMethod: string;
	VerifiedAt: string | null;
	LastLinkedAt: string | null;
	RetiredAt: string | null;
	CreatedAt: string;
}
