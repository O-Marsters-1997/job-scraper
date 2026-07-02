export interface SourceFilterField {
	name: string;
	label: string;
	required: boolean;
}

export interface SourceInfo {
	name: string;
	label: string;
	kind: "board" | "url" | "filter";
	role: "ats" | "discovery";
	url_prefix: string;
	filters: SourceFilterField[];
}
