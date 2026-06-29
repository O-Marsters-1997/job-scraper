export interface JobScore {
	JobID: string;
	UserID: string;
	RelevanceScore: number | null;
	SuitabilityScore: number | null;
	Reasoning: string | null;
	Matched: string[] | null;
	Missing: string[] | null;
	SuitabilitySkipped: boolean;
}
