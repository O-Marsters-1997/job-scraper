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

export function orderSuggestions(
	suggestions: Suggestion[],
	order: string[],
): Suggestion[] {
	const rank = new Map(order.map((id, i) => [id, i]));
	return [...suggestions].sort(
		(a, b) =>
			(rank.get(a.achievementId) ?? order.length) -
			(rank.get(b.achievementId) ?? order.length),
	);
}

export function selectedAchievementIds(
	suggestions: Suggestion[],
	overrides: Record<string, boolean>,
	order: string[],
): string[] {
	return orderSuggestions(suggestions, order)
		.filter((s) => overrides[s.achievementId] ?? s.preselected)
		.map((s) => s.achievementId);
}

export class KeptDraftExistsError extends Error {
	constructor() {
		super("This job already has a kept draft");
	}
}

function isLegacySkillGap(f: DraftFinding): boolean {
	return f.check === "skills" && f.severity === "info";
}

const SEVERITY_ORDER: Record<DraftFinding["severity"], number> = {
	block: 0,
	warn: 1,
	info: 2,
};

export function reviewFindings(findings: DraftFinding[]): DraftFinding[] {
	return findings
		.filter((f) => !isLegacySkillGap(f))
		.sort((a, b) => SEVERITY_ORDER[a.severity] - SEVERITY_ORDER[b.severity]);
}

export function keptDraft<T extends { outcome: string | null }>(
	drafts: T[],
): T | undefined {
	return drafts.find((d) => d.outcome === "kept");
}

export function lineWithIds(current: string[], ids: string[]): string[] {
	const keep = current.filter((item) => ids.includes(item));
	return [...keep, ...ids.filter((id) => !current.includes(id))];
}

export function moveItem(items: string[], from: number, to: number): string[] {
	if (to < 0 || to >= items.length || from === to) return items;
	const next = [...items];
	const [moved] = next.splice(from, 1);
	if (moved !== undefined) next.splice(to, 0, moved);
	return next;
}
