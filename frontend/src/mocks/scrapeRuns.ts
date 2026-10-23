import type { ScrapeRun, ScrapeRunPage, SourceHealth } from "@/types/scrapeRun";

const PAGE_SIZE = 50;
const TOTAL = 120;
const NEWEST = Date.parse("2026-10-23T16:08:00Z");

const TARGETS = [
	{ Source: "greenhouse", TargetValue: "monzo" },
	{ Source: "linkedin", TargetValue: "Senior Go Engineer" },
	{ Source: "lever", TargetValue: "bud" },
	{ Source: "indeed", TargetValue: "Backend Developer" },
] as const;

const SHAPES = [
	{
		Outcome: "succeeded",
		StopReason: "last_page",
		Retries: 0,
		New: 3,
		Listings: 20,
	},
	{
		Outcome: "succeeded",
		StopReason: "page_cap",
		Retries: 0,
		New: 9,
		Listings: 40,
	},
	{
		Outcome: "retried",
		StopReason: "last_page",
		Retries: 1,
		New: 1,
		Listings: 20,
	},
	{ Outcome: "failed", StopReason: "failed", Retries: 3, New: 0, Listings: 0 },
] as const;

function mockRun(i: number): ScrapeRun {
	const startedAt = NEWEST - i * 17 * 60_000;
	const shape = SHAPES[i % SHAPES.length] ?? SHAPES[0];
	const target = TARGETS[i % TARGETS.length] ?? TARGETS[0];
	const running = i === 0;
	return {
		ID: `run-${String(i).padStart(3, "0")}`,
		Kind: "search",
		...target,
		TargetID: `target-${i % TARGETS.length}`,
		TargetFilters: {},
		Trigger: i % 11 === 5 ? "manual" : "scheduled",
		...shape,
		...(running && { Outcome: "running", StopReason: "" }),
		StartedAt: new Date(startedAt).toISOString(),
		FinishedAt: running ? null : new Date(startedAt + 15_000).toISOString(),
		Pages: shape.Listings > 0 ? 2 : 0,
		Elsewhere: i % 4,
		Repeat: 12,
		Deferred: 0,
		Errors:
			shape.Retries > 0
				? [
						{
							Page: 1,
							Attempt: 1,
							Message: "parse listing cards: no .job-card nodes found",
							At: new Date(startedAt + 5_000).toISOString(),
						},
					]
				: [],
	};
}

const RUNS = Array.from({ length: TOTAL }, (_, i) => mockRun(i));

export function getScrapeRuns(params: {
	target?: string | undefined;
	before?: string | undefined;
}): ScrapeRunPage {
	const matching = RUNS.filter(
		(r) => !params.target || r.TargetID === params.target,
	);
	const start = params.before
		? matching.findIndex((r) => r.ID === params.before) + 1
		: 0;
	const items = matching.slice(start, start + PAGE_SIZE);
	const last = items[items.length - 1];
	const more = start + PAGE_SIZE < matching.length;
	return { Items: items, NextCursor: more && last ? last.ID : null };
}

export function getSourceHealth(): SourceHealth[] {
	return TARGETS.map((target, n): SourceHealth => {
		const targetRuns = RUNS.filter(
			(r) => r.TargetID === `target-${n}` && r.FinishedAt,
		);
		const last5 = targetRuns.slice(0, 5).reverse();
		const recent = last5.map((r) => r.Outcome);
		const sum = (pick: (r: ScrapeRun) => number) =>
			last5.reduce((total, r) => total + pick(r), 0);
		const retries = sum((r) => r.Retries);
		const classify = (): [SourceHealth["Status"], SourceHealth["Reason"]] => {
			if (recent.length === 0) return ["green", "no_runs"];
			if (recent.at(-1) === "failed") return ["red", "last_failed"];
			if (retries > 0) return ["amber", "retried"];
			return ["green", "ok"];
		};
		const [status, reason] = classify();
		return {
			Kind: "search",
			TargetID: `target-${n}`,
			...target,
			TargetFilters: {},
			Status: status,
			Reason: reason,
			Recent: recent,
			LastRunAt: last5.at(-1)?.StartedAt ?? null,
			New: sum((r) => r.New),
			Elsewhere: sum((r) => r.Elsewhere),
			Retries: retries,
		};
	});
}
