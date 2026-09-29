import type { CVHeading, HeadingMapping } from "../types/tailoring";

export function allConfirmed(headings: CVHeading[]): boolean {
	return headings.every((h) => h.confirmed);
}

export function toMappings(
	headings: CVHeading[],
	chosen: Record<string, string | null>,
): HeadingMapping[] {
	return headings.map((h) => ({
		headingText: h.text,
		positionId: h.text in chosen ? (chosen[h.text] ?? null) : h.positionId,
	}));
}
