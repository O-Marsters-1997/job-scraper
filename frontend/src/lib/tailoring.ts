import type {
	CVHeading,
	DraftFinding,
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

export class KeptDraftExistsError extends Error {
	constructor() {
		super("This job already has a kept draft");
	}
}

function isSkillGap(f: DraftFinding): boolean {
	return f.check === "skills" && f.severity === "info";
}

const SEVERITY_ORDER: Record<DraftFinding["severity"], number> = {
	block: 0,
	warn: 1,
	info: 2,
};

export function reviewFindings(findings: DraftFinding[]): DraftFinding[] {
	return findings
		.filter((f) => !isSkillGap(f))
		.sort((a, b) => SEVERITY_ORDER[a.severity] - SEVERITY_ORDER[b.severity]);
}

export function skillGaps(findings: DraftFinding[]): string[] {
	return findings.filter(isSkillGap).map((f) => f.message);
}

export function keptDraft<T extends { outcome: string | null }>(
	drafts: T[],
): T | undefined {
	return drafts.find((d) => d.outcome === "kept");
}
