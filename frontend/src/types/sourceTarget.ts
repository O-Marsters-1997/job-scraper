export interface SourceTarget {
	ID: string;
	UserID: string;
	Source: string;
	Value: string;
	Enabled: boolean;
	Filters: Record<string, string>;
}
