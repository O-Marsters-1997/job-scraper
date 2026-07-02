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
