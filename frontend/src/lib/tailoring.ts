import type { DraftFinding, DraftStatus } from "../types/tailoring";

export function isSettled(status: DraftStatus): boolean {
	return status === "ready" || status === "failed";
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
