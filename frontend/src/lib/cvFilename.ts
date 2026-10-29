export const DEFAULT_CV_FILENAME_PATTERN = "cv-{company}";

type CvFilenameToken = "name" | "company" | "role" | "date";

export type CvFilenameValues = Record<CvFilenameToken, string>;

function clean(text: string): string {
	return text
		.replace(/[\\/:*?"<>|]|\p{Cc}/gu, "-")
		.replace(/\s+/g, "-")
		.replace(/-{2,}/g, "-");
}

export function cvFilename(pattern: string, values: CvFilenameValues): string {
	const filled = (pattern.trim() || DEFAULT_CV_FILENAME_PATTERN).replace(
		/\{(\w+)\}/g,
		(token, key: string) =>
			key in values ? values[key as CvFilenameToken] : token,
	);
	const base = clean(filled).replace(/^[-.]+|[-.]+$/g, "");
	return `${base || "cv"}.pdf`;
}
