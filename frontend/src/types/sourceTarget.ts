export interface SourceTarget {
	ID: string;
	UserID: string;
	Source: string;
	Value: string;
	Enabled: boolean;
	Filters: Record<string, string>;
	RunStatus: "idle" | "queued" | "running" | "succeeded" | "failed";
	LastRunAt: string | null;
	LastRunError: string;
}
