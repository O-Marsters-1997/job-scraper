import type {
	CVHeading,
	DraftFinding,
	DraftStatus,
	HeadingMapping,
	SkillCandidate,
	SkillLineSuggestion,
	SkillPick,
	SkillSuggestions,
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

export function canExplain(s: Suggestion): boolean {
	return s.state !== "fit";
}

export function moveSuggestion(
	ordered: Suggestion[],
	id: string,
	to: number,
): string[] {
	const moved = ordered.find((s) => s.achievementId === id);
	const ids = ordered.map((s) => s.achievementId);
	if (!moved) return ids;
	const slots = ordered.flatMap((s, i) =>
		s.positionId === moved.positionId ? [i] : [],
	);
	const group = slots.flatMap((i) => ids[i] ?? []);
	const from = group.indexOf(id);
	if (to < 0 || to >= group.length) return ids;
	group.splice(to, 0, ...group.splice(from, 1));
	for (const [n, slot] of slots.entries()) {
		const next = group[n];
		if (next !== undefined) ids[slot] = next;
	}
	return ids;
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

export function preselectedPicks(data: SkillSuggestions): SkillPick[] {
	return data.lines.flatMap((l, line) =>
		l.candidates
			.filter((c) => c.preselected)
			.map((c) => ({ bankSkillId: c.bankSkillId, line, replaces: c.replaces })),
	);
}

export function nextReplaces(
	line: SkillLineSuggestion,
	lineIndex: number,
	picks: SkillPick[],
): string {
	const taken = new Set(
		picks.filter((p) => p.line === lineIndex).map((p) => p.replaces),
	);
	return (
		line.base
			.filter((b) => !taken.has(b.text))
			.sort((a, b) => a.score - b.score)[0]?.text ?? ""
	);
}

export function candidatesFor(
	data: SkillSuggestions,
	lineIndex: number,
	picks: SkillPick[],
): SkillCandidate[] {
	const placed = new Set(
		picks.filter((p) => p.line === lineIndex).map((p) => p.bankSkillId),
	);
	const own = data.lines[lineIndex]?.candidates ?? [];
	return [...own, ...data.unplaced.filter((c) => placed.has(c.bankSkillId))];
}

export function setLinePicks(
	data: SkillSuggestions,
	picks: SkillPick[],
	lineIndex: number,
	ids: string[],
): SkillPick[] {
	const line = data.lines[lineIndex];
	if (!line) return picks;
	const rest = picks.filter((p) => p.line !== lineIndex);
	const kept = picks.filter(
		(p) => p.line === lineIndex && ids.includes(p.bankSkillId),
	);
	const next = [...kept];
	for (const id of ids) {
		if (kept.some((p) => p.bankSkillId === id)) continue;
		const replaces = nextReplaces(line, lineIndex, [...rest, ...next]);
		if (replaces) next.push({ bankSkillId: id, line: lineIndex, replaces });
	}
	return [...rest, ...next];
}

export function placeUnplaced(
	data: SkillSuggestions,
	picks: SkillPick[],
	bankSkillId: string,
	lineIndex: number | undefined,
): SkillPick[] {
	const rest = picks.filter((p) => p.bankSkillId !== bankSkillId);
	const line = lineIndex === undefined ? undefined : data.lines[lineIndex];
	if (lineIndex === undefined || !line) return rest;
	return [
		...rest,
		{
			bankSkillId,
			line: lineIndex,
			replaces: nextReplaces(line, lineIndex, rest),
		},
	];
}

export function retarget(
	picks: SkillPick[],
	bankSkillId: string,
	replaces: string,
): SkillPick[] {
	return picks.map((p) =>
		p.bankSkillId === bankSkillId ? { ...p, replaces } : p,
	);
}
