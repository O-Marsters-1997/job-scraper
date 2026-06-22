export interface SourceFilterField {
	name: string;
	label: string;
	required: boolean;
}

export interface SourceInfo {
	name: string;
	label: string;
	kind: "board" | "url" | "filter";
	url_prefix: string;
	filters: SourceFilterField[];
}
