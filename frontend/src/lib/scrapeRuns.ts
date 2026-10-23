export interface ActivitySearch {
	target?: string | undefined;
	before?: string | undefined;
	trail?: string | undefined;
}

export function parseActivitySearch(
	raw: Record<string, unknown>,
): ActivitySearch {
	const text = (v: unknown) => (typeof v === "string" && v ? v : undefined);
	const before = text(raw.before);
	return {
		target: text(raw.target),
		before,
		trail: before ? text(raw.trail) : undefined,
	};
}

export function olderPage(
	search: ActivitySearch,
	next: string,
): ActivitySearch {
	const trail = [search.trail, search.before].filter(Boolean).join(",");
	return { ...search, before: next, trail: trail || undefined };
}

export function newerPage(search: ActivitySearch): ActivitySearch {
	const trail = search.trail?.split(",") ?? [];
	const before = trail.pop();
	return { ...search, before, trail: trail.join(",") || undefined };
}

export function formatDuration(startedAt: string, finishedAt: string): string {
	const seconds = Math.round(
		(Date.parse(finishedAt) - Date.parse(startedAt)) / 1000,
	);
	if (seconds < 60) return `${seconds}s`;
	return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}
