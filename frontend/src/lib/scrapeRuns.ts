import type { SourceHealth } from "@/types/scrapeRun";

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

const STATUS_RANK = { red: 0, amber: 1, green: 2 } as const;

export function splitHealth(rows: SourceHealth[]): {
	attention: SourceHealth[];
	healthy: SourceHealth[];
} {
	const lastRun = (r: SourceHealth) =>
		r.LastRunAt ? Date.parse(r.LastRunAt) : 0;
	const byWorst = [...rows].sort(
		(a, b) =>
			STATUS_RANK[a.Status] - STATUS_RANK[b.Status] || lastRun(b) - lastRun(a),
	);
	return {
		attention: byWorst.filter((r) => r.Status !== "green"),
		healthy: byWorst.filter((r) => r.Status === "green"),
	};
}

export function healthLabel(row: SourceHealth): string {
	switch (row.Reason) {
		case "last_failed":
			return "Last run failed";
		case "failing_streak":
			return "Failing repeatedly";
		case "retried":
			return `${row.Retries} retried ${row.Retries === 1 ? "attempt" : "attempts"}`;
		case "no_new":
			return "Nothing new in 5 runs";
		case "no_runs":
			return "No runs yet";
		case "ok":
			return "Healthy";
	}
}
