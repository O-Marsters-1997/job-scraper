import type {
	CVHeading,
	DraftStatus,
	HeadingMapping,
	Suggestion,
} from "../types/tailoring";

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

export function isSettled(status: DraftStatus): boolean {
	return status === "ready" || status === "failed";
}

export function selectedAchievementIds(
	suggestions: Suggestion[],
	overrides: Record<string, boolean>,
): string[] {
	return suggestions
		.filter((s) => overrides[s.achievementId] ?? s.preselected)
		.map((s) => s.achievementId);
}
