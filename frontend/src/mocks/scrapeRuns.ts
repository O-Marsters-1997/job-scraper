import type { ScrapeRunFilters, ScrapeRunsParams } from "@/api/scrapeRuns";
import type {
	ScrapeRun,
	ScrapeRunFacet,
	ScrapeRunPage,
	ScrapeRunSummary,
	SourceHealth,
} from "@/types/scrapeRun";

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
		BoardID: "",
		CompanyName: "",
		Removed: 0,
		Reopened: 0,
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

const RANGE_MS: Record<string, number> = {
	"1h": 3_600_000,
	"6h": 6 * 3_600_000,
	"24h": 24 * 3_600_000,
	"7d": 7 * 24 * 3_600_000,
	"30d": 30 * 24 * 3_600_000,
};
const BUCKET_MS = [5, 15, 60, 180, 360, 720, 1440].map((m) => m * 60_000);

const FACET_VALUE: Record<ScrapeRunFacet, (r: ScrapeRun) => string> = {
	outcome: (r) => r.Outcome,
	kind: (r) => r.Kind,
	source: (r) => r.Source,
	trigger: (r) => r.Trigger,
};

function rangeStart(params: ScrapeRunFilters): number {
	if (params.from) return Date.parse(params.from);
	return NEWEST - (RANGE_MS[params.range ?? "7d"] ?? RANGE_MS["7d"] ?? 0);
}

function inFilters(
	run: ScrapeRun,
	params: ScrapeRunFilters,
	skip?: ScrapeRunFacet,
): boolean {
	const started = Date.parse(run.StartedAt);
	const to = params.to ? Date.parse(params.to) : Number.POSITIVE_INFINITY;
	const needle = params.q?.toLowerCase();
	return (
		(!params.target || run.TargetID === params.target) &&
		!params.company &&
		started >= rangeStart(params) &&
		started <= to &&
		(!needle ||
			run.TargetValue.toLowerCase().includes(needle) ||
			run.Errors.some((e) => e.Message.toLowerCase().includes(needle))) &&
		(Object.keys(FACET_VALUE) as ScrapeRunFacet[]).every((facet) => {
			const selected = params[facet]?.split(",") ?? [];
			return (
				facet === skip ||
				selected.length === 0 ||
				selected.includes(FACET_VALUE[facet](run))
			);
		})
	);
}

function rollup(runs: ScrapeRun[]) {
	return {
		New: runs.reduce((n, r) => n + r.New, 0),
		Removed: runs.reduce((n, r) => n + r.Removed, 0),
		LastStartedAt: runs[0]?.StartedAt ?? null,
	};
}

export function getScrapeRunSummary(
	params: ScrapeRunFilters,
): ScrapeRunSummary {
	const end = params.to ? Date.parse(params.to) : NEWEST;
	const from = rangeStart(params);
	const bucket =
		BUCKET_MS.find((b) => end - from <= 48 * b) ?? BUCKET_MS[6] ?? 86_400_000;
	const facets: ScrapeRunSummary["Facets"] = {
		outcome: {},
		kind: {},
		source: {},
		trigger: {},
	};
	for (const facet of Object.keys(FACET_VALUE) as ScrapeRunFacet[]) {
		const counts = facets[facet] ?? {};
		for (const run of RUNS.filter((r) => inFilters(r, params, facet))) {
			const value = FACET_VALUE[facet](run);
			counts[value] = (counts[value] ?? 0) + 1;
		}
	}
	const first = Math.floor(from / bucket) * bucket;
	const histogram = Array.from(
		{ length: Math.floor((end - first) / bucket) + 1 },
		(_, i) => ({
			Start: new Date(first + i * bucket).toISOString(),
			OK: 0,
			Retried: 0,
			Failed: 0,
		}),
	);
	const matching = RUNS.filter((r) => inFilters(r, params));
	for (const run of matching) {
		const slot =
			histogram[Math.floor((Date.parse(run.StartedAt) - first) / bucket)];
		if (!slot) continue;
		if (run.Outcome === "failed") slot.Failed++;
		else if (run.Outcome === "retried") slot.Retried++;
		else slot.OK++;
	}
	return {
		Total: matching.length,
		...rollup(matching),
		Facets: facets,
		BucketMs: bucket,
		Histogram: histogram,
	};
}

export function getScrapeRuns(params: ScrapeRunsParams): ScrapeRunPage {
	const matching = RUNS.filter((r) => inFilters(r, params));
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
			BoardID: "",
			CompanyName: "",
			Status: status,
			Reason: reason,
			Recent: recent,
			LastRunAt: last5.at(-1)?.StartedAt ?? null,
			New: sum((r) => r.New),
			Elsewhere: sum((r) => r.Elsewhere),
			Removed: 0,
			Retries: retries,
		};
	});
}
