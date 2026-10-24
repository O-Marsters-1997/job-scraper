import { Popover } from "@kobalte/core/popover";
import { createQuery, keepPreviousData } from "@tanstack/solid-query";
import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, type JSX, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { SearchField } from "@/components/SearchField";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { SkeletonList } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { formatRelative } from "@/lib/datetime";
import { cn } from "@/lib/utils";

type Kind = "search" | "board";
type Health = "red" | "amber" | "green";
type Outcome = "failed" | "errored" | "ok" | "running";
type Flavour = "failing" | "flaky" | "stale" | "capped" | "recovered";

interface Target {
	id: string;
	kind: Kind;
	source: string;
	name: string;
}

interface RunError {
	page: number;
	attempt: number;
	message: string;
	at: string;
}

interface Run {
	id: string;
	target: Target;
	startedAt: string;
	durationSec: number;
	status: "running" | "succeeded" | "failed";
	trigger: "scheduled" | "manual";
	stopReason: "last_page" | "page_cap" | "failed" | null;
	pages: number;
	listed: number;
	fresh: number;
	elsewhere: number;
	repeat: number;
	removed: number;
	deferred: number;
	errors: RunError[];
}

const HOUR = 3600_000;
const DAY = 24 * HOUR;
const RETENTION_DAYS = 30;

const SEARCHES: [string, string, number, Flavour?][] = [
	["LinkedIn", "Senior Go Engineer · London", 6, "capped"],
	["LinkedIn", "Platform Engineer · Remote UK", 6],
	["LinkedIn", "Staff Engineer · Bristol", 12],
	["LinkedIn", "Backend Engineer · Leeds", 12],
	["Indeed", "Backend Developer · Manchester", 12, "failing"],
	["Indeed", "Go Developer · Remote", 24],
	["Indeed", "Software Engineer · Edinburgh", 24, "stale"],
	["hiring.cafe", "Software Engineer", 12, "stale"],
	["hiring.cafe", "Infrastructure Engineer", 12],
	["Wellfound", "Founding Engineer", 24],
	["Wellfound", "Backend Engineer · Seed stage", 24, "recovered"],
	["Wellfound", "Engineering Lead", 24],
];

const BOARDS: [string, string, Flavour?][] = [
	["Greenhouse", "Monzo"],
	["Greenhouse", "Cleo", "flaky"],
	["Greenhouse", "Deliveroo"],
	["Greenhouse", "GoCardless"],
	["Greenhouse", "Wise"],
	["Greenhouse", "Thought Machine"],
	["Greenhouse", "Paddle", "recovered"],
	["Greenhouse", "Tide"],
	["Lever", "Octopus Energy"],
	["Lever", "Zopa"],
	["Lever", "Cuvva"],
	["Lever", "Marshmallow", "stale"],
	["Ashby", "Synthesia", "failing"],
	["Ashby", "Lindus Health"],
	["Ashby", "Ravelin"],
	["Ashby", "PhysicsX"],
	["Ashby", "Tessl"],
	["Workable", "Bloom & Wild", "stale"],
	["Workable", "Hopin"],
	["Workable", "Huel"],
	["Workable", "Papier"],
	["Greenhouse", "Moneybox"],
	["Lever", "Bud"],
	["Ashby", "Granola"],
	["Workable", "Pact Coffee", "flaky"],
	["Greenhouse", "Multiverse"],
	["Lever", "Snyk"],
	["Ashby", "Cera"],
];

const TARGETS: (Target & { flavour?: Flavour; intervalHours: number })[] = [
	...SEARCHES.map(([source, name, intervalHours, flavour], i) => ({
		id: `s${i}`,
		kind: "search" as const,
		source,
		name,
		intervalHours,
		...(flavour ? { flavour } : {}),
	})),
	...BOARDS.map(([source, name, flavour], i) => ({
		id: `b${i}`,
		kind: "board" as const,
		source,
		name,
		intervalHours: 6,
		...(flavour ? { flavour } : {}),
	})),
];

const ERROR_MESSAGES = {
	unavailable: "GET page: 503 Service Unavailable",
	parse: "parse listing cards: no .job-card nodes found",
	timeout: "context deadline exceeded after 30s",
	gone: "GET board: 404 Not Found (token may have changed)",
	rateLimit: "GET page: 429 Too Many Requests",
};

function seeded(seed: number) {
	let s = seed;
	return () => {
		s = (s * 1664525 + 1013904223) % 4294967296;
		return s / 4294967296;
	};
}

const LINKEDIN_OUTAGE = {
	from: Date.now() - 9 * DAY,
	to: Date.now() - 8.2 * DAY,
};

function makeRuns(): Run[] {
	const rand = seeded(42);
	const int = (lo: number, hi: number) =>
		lo + Math.floor(rand() * (hi - lo + 1));
	const pick = <T,>(xs: T[]) => xs[int(0, xs.length - 1)] as T;
	const now = Date.now();
	const runs: Run[] = [];
	for (const t of TARGETS) {
		let start = now - rand() * t.intervalHours * HOUR;
		for (let i = 0; start > now - RETENTION_DAYS * DAY; i++) {
			const outage =
				t.source === "LinkedIn" &&
				start > LINKEDIN_OUTAGE.from &&
				start < LINKEDIN_OUTAGE.to;
			const failing =
				outage ||
				(t.flavour === "failing" && i < 3) ||
				(t.flavour === "recovered" && i >= 6 && i < 9) ||
				rand() < 0.01;
			const flaky = t.flavour === "flaky" ? rand() < 0.6 : rand() < 0.015;
			const running = i === 0 && t.id === "s1";
			const pages =
				t.kind === "board"
					? int(1, 2)
					: t.flavour === "capped"
						? 10
						: int(2, 9);
			const at = (sec: number) => new Date(start + sec * 1000).toISOString();
			const errors: RunError[] = [];
			if (failing) {
				const message = outage
					? ERROR_MESSAGES.rateLimit
					: t.kind === "board"
						? ERROR_MESSAGES.gone
						: ERROR_MESSAGES.parse;
				for (let a = 1; a <= 3; a++)
					errors.push({ page: 1, attempt: a, message, at: at(a * 20) });
			} else if (flaky && !running) {
				errors.push({
					page: int(1, pages),
					attempt: 1,
					message: pick([
						ERROR_MESSAGES.unavailable,
						ERROR_MESSAGES.timeout,
						ERROR_MESSAGES.parse,
					]),
					at: at(int(5, 60)),
				});
			}
			const listed = failing
				? 0
				: t.kind === "board"
					? int(12, 60)
					: pages * 25;
			const fresh =
				failing || t.flavour === "stale" ? 0 : int(0, Math.ceil(listed / 6));
			const elsewhere = failing ? 0 : int(0, Math.ceil((listed - fresh) / 3));
			runs.push({
				id: `${t.id}-${String(i).padStart(4, "0")}`,
				target: t,
				startedAt: new Date(start).toISOString(),
				durationSec: running
					? 0
					: t.kind === "board"
						? int(2, 15)
						: pages * int(8, 25),
				status: running ? "running" : failing ? "failed" : "succeeded",
				trigger: rand() < 0.06 ? "manual" : "scheduled",
				stopReason: running
					? null
					: failing
						? "failed"
						: t.flavour === "capped"
							? "page_cap"
							: "last_page",
				pages: failing ? 1 : pages,
				listed,
				fresh,
				elsewhere,
				repeat: listed - fresh - elsewhere,
				removed: t.kind === "board" && !failing ? int(0, 3) : 0,
				deferred: t.source === "LinkedIn" ? int(0, 2) : 0,
				errors,
			});
			start -= t.intervalHours * HOUR * (0.9 + rand() * 0.2);
		}
	}
	return runs.sort(byNewest);
}

function byNewest(a: Run, b: Run) {
	return b.startedAt.localeCompare(a.startedAt) || b.id.localeCompare(a.id);
}

const RUNS = makeRuns();

function outcomeOf(r: Run): Outcome {
	if (r.status === "running") return "running";
	if (r.status === "failed") return "failed";
	return r.errors.length > 0 ? "errored" : "ok";
}

function lastFive(targetId: string) {
	return RUNS.filter(
		(r) => r.target.id === targetId && r.status !== "running",
	).slice(0, 5);
}

function healthOf(targetId: string): Health {
	const last5 = lastFive(targetId);
	const failed = last5.map((r) => r.status === "failed");
	if (failed[0] || failed.some((f, i) => f && failed[i + 1])) return "red";
	if (
		last5.some((r) => r.errors.length > 0) ||
		last5.every((r) => r.fresh === 0)
	)
		return "amber";
	return "green";
}

function healthReason(targetId: string): string {
	const last5 = lastFive(targetId);
	switch (healthOf(targetId)) {
		case "red":
			return last5[0]?.status === "failed"
				? "Last run failed"
				: "Failed twice in a row";
		case "amber": {
			if (last5.every((r) => r.fresh === 0)) return "Nothing new in 5 runs";
			const n = last5.reduce((sum, r) => sum + r.errors.length, 0);
			return `${n} retried ${n === 1 ? "attempt" : "attempts"}`;
		}
		default:
			return "Healthy";
	}
}

const HEALTH_ORDER: Record<Health, number> = { red: 0, amber: 1, green: 2 };
const OUTCOME_DOT: Record<Outcome, string> = {
	failed: "bg-destructive",
	errored: "bg-status-interview",
	ok: "bg-status-offer",
	running: "bg-primary",
};
const OUTCOME_LABEL: Record<Outcome, string> = {
	failed: "Failed",
	errored: "Retried",
	ok: "Succeeded",
	running: "Running",
};
const HEALTH_DOT: Record<Health, string> = {
	red: OUTCOME_DOT.failed,
	amber: OUTCOME_DOT.errored,
	green: OUTCOME_DOT.ok,
};

type FacetKey = "outcome" | "kind" | "source" | "trigger";

interface Facet {
	key: FacetKey;
	label: string;
	values: { value: string; label: string }[];
	of: (r: Run) => string;
}

const FACETS: Facet[] = [
	{
		key: "outcome",
		label: "Outcome",
		values: (["failed", "errored", "ok", "running"] as Outcome[]).map((o) => ({
			value: o,
			label: OUTCOME_LABEL[o],
		})),
		of: outcomeOf,
	},
	{
		key: "kind",
		label: "Type",
		values: [
			{ value: "search", label: "Searches" },
			{ value: "board", label: "Boards" },
		],
		of: (r) => r.target.kind,
	},
	{
		key: "source",
		label: "Source",
		values: [...new Set(TARGETS.map((t) => t.source))].map((s) => ({
			value: s,
			label: s,
		})),
		of: (r) => r.target.source,
	},
	{
		key: "trigger",
		label: "Trigger",
		values: [
			{ value: "scheduled", label: "Scheduled" },
			{ value: "manual", label: "Manual" },
		],
		of: (r) => r.trigger,
	},
];

const PRESETS = [
	{ value: "1h", label: "Last hour", ms: HOUR },
	{ value: "6h", label: "Last 6 hours", ms: 6 * HOUR },
	{ value: "24h", label: "Last 24 hours", ms: DAY },
	{ value: "7d", label: "Last 7 days", ms: 7 * DAY },
	{ value: "30d", label: "Last 30 days", ms: 30 * DAY },
] as const;
const DEFAULT_RANGE = "7d";

interface RunQuery {
	q?: string | undefined;
	outcome?: string[] | undefined;
	kind?: string[] | undefined;
	source?: string[] | undefined;
	trigger?: string[] | undefined;
	target?: string | undefined;
	range?: string | undefined;
	from?: string | undefined;
	to?: string | undefined;
}

interface RunsPage {
	items: Run[];
	nextCursor: string | null;
}

interface RunsSummary {
	total: number;
	facets: Record<FacetKey, Record<string, number>>;
	bucketMs: number;
	histogram: { start: number; ok: number; errored: number; failed: number }[];
}

const PAGE_LIMIT = 50;
const BUCKET_STEPS = [
	5 * 60_000,
	15 * 60_000,
	HOUR,
	3 * HOUR,
	6 * HOUR,
	12 * HOUR,
	DAY,
];

function resolveWindow(q: RunQuery, now = Date.now()) {
	if (q.from && q.to) return { from: Date.parse(q.from), to: Date.parse(q.to) };
	const preset = PRESETS.find((p) => p.value === q.range) ?? PRESETS[3];
	return { from: now - preset.ms, to: now };
}

function matches(
	r: Run,
	q: RunQuery,
	window: { from: number; to: number },
	except?: FacetKey,
) {
	const t = Date.parse(r.startedAt);
	if (t < window.from || t > window.to) return false;
	if (q.target && r.target.id !== q.target) return false;
	if (q.q) {
		const needle = q.q.toLowerCase();
		const hay = [
			r.target.name,
			r.target.source,
			...r.errors.map((e) => e.message),
		]
			.join(" ")
			.toLowerCase();
		if (!hay.includes(needle)) return false;
	}
	return FACETS.every((f) => {
		const picked = q[f.key];
		return f.key === except || !picked || picked.includes(f.of(r));
	});
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const encodeCursor = (r: Run) => btoa(`${r.startedAt}|${r.id}`);

async function fetchRunsPage(
	q: RunQuery,
	before: string | undefined,
): Promise<RunsPage> {
	await sleep(220);
	const window = resolveWindow(q);
	let rows = RUNS.filter((r) => matches(r, q, window));
	if (before) {
		const [ts = "", id = ""] = atob(before).split("|");
		rows = rows.filter(
			(r) => r.startedAt < ts || (r.startedAt === ts && r.id < id),
		);
	}
	const items = rows.slice(0, PAGE_LIMIT);
	const last = items.at(-1);
	return {
		items,
		nextCursor: rows.length > PAGE_LIMIT && last ? encodeCursor(last) : null,
	};
}

async function fetchRunsSummary(q: RunQuery): Promise<RunsSummary> {
	await sleep(180);
	const window = resolveWindow(q);
	const span = window.to - window.from;
	const bucketMs = BUCKET_STEPS.find((s) => span / s <= 48) ?? DAY;
	const first = Math.floor(window.from / bucketMs) * bucketMs;
	const histogram = Array.from(
		{ length: Math.ceil((window.to - first) / bucketMs) },
		(_, i) => ({
			start: first + i * bucketMs,
			ok: 0,
			errored: 0,
			failed: 0,
		}),
	);
	const facets = Object.fromEntries(
		FACETS.map((f) => [f.key, {}]),
	) as RunsSummary["facets"];
	let total = 0;
	for (const r of RUNS) {
		for (const f of FACETS) {
			if (!matches(r, q, window, f.key)) continue;
			const counts = facets[f.key];
			counts[f.of(r)] = (counts[f.of(r)] ?? 0) + 1;
		}
		if (!matches(r, q, window)) continue;
		total++;
		const b =
			histogram[Math.floor((Date.parse(r.startedAt) - first) / bucketMs)];
		const o = outcomeOf(r);
		if (!b) continue;
		if (o === "failed") b.failed++;
		else if (o === "errored") b.errored++;
		else b.ok++;
	}
	return { total, facets, bucketMs, histogram };
}

interface ActivitySearch extends RunQuery {
	before?: string | undefined;
}

type Patch = Partial<ActivitySearch>;

const list = (v: unknown) => {
	const arr = (Array.isArray(v) ? v : [v]).filter(
		(x): x is string => typeof x === "string" && x !== "",
	);
	return arr.length ? arr : undefined;
};
const str = (v: unknown) => (typeof v === "string" && v ? v : undefined);

export const Route = createFileRoute("/_auth/activity-prototype")({
	validateSearch: (s: Record<string, unknown>): ActivitySearch => ({
		q: str(s.q),
		outcome: list(s.outcome),
		kind: list(s.kind),
		source: list(s.source),
		trigger: list(s.trigger),
		target: str(s.target),
		range: str(s.range),
		from: str(s.from),
		to: str(s.to),
		before: str(s.before),
	}),
	component: ActivityPage,
});

const CLEARED: Patch = {
	q: undefined,
	outcome: undefined,
	kind: undefined,
	source: undefined,
	trigger: undefined,
	target: undefined,
};

function ActivityPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/activity-prototype" });
	const [cursors, setCursors] = createSignal<(string | undefined)[]>([]);
	const [live, setLive] = createSignal(true);
	const [facetsOpen, setFacetsOpen] = createSignal(false);
	const go = (patch: Patch) =>
		navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true });
	const set = (patch: Patch) => {
		setCursors([]);
		go({ before: undefined, ...patch });
	};
	const runQuery = (): RunQuery => {
		const { before: _, ...q } = search();
		return q;
	};
	const isNewest = () => !search().before;

	const page = createQuery(() => ({
		queryKey: ["scrape-runs", runQuery(), search().before],
		queryFn: () => fetchRunsPage(runQuery(), search().before),
		placeholderData: keepPreviousData,
		refetchInterval: live() && isNewest() ? 5000 : false,
	}));
	const summary = createQuery(() => ({
		queryKey: ["scrape-runs-summary", runQuery()],
		queryFn: () => fetchRunsSummary(runQuery()),
		placeholderData: keepPreviousData,
		refetchInterval: live() && isNewest() ? 5000 : false,
	}));
	const activeFacets = () => FACETS.filter((f) => search()[f.key]).length;
	let logEl: HTMLElement | undefined;

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Activity"
				subtitle="Scrape runs for your searches and tracked boards, kept for 30 days"
			/>
			<HealthTable
				selected={search().target}
				onSelect={(id) => {
					set({ target: id });
					if (id) logEl?.scrollIntoView({ behavior: "smooth", block: "start" });
				}}
			/>
			<section ref={logEl} class="mt-8 scroll-mt-6">
				<h2 class="mb-3 text-base font-semibold text-foreground">Run log</h2>
				<div class="mb-3 flex flex-wrap items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						aria-expanded={facetsOpen()}
						aria-controls="run-facets"
						onClick={() => setFacetsOpen(!facetsOpen())}
						class={cn(
							facetsOpen() &&
								"border-accent-border bg-accent-subtle text-accent-text",
						)}
					>
						<Icon name="sliders" size={14} />
						Filters
						<Show when={activeFacets() > 0}>
							<span class="rounded-full bg-primary px-1.5 font-mono text-2xs leading-4 text-primary-foreground">
								{activeFacets()}
							</span>
						</Show>
					</Button>
					<SearchField
						id="run-search"
						label="Search runs"
						placeholder="Search names and error messages"
						value={search().q ?? ""}
						onInput={(q) => set({ q: q || undefined })}
						class="sm:max-w-72"
					/>
					<div class="ml-auto flex items-center gap-2">
						<span class="text-xs text-faint">
							<span class="font-mono tabular-nums text-foreground">
								{summary.data?.total.toLocaleString() ?? "–"}
							</span>{" "}
							runs
						</span>
						<DateRangePicker
							search={search()}
							onChange={(r) => set({ range: r.range, from: r.from, to: r.to })}
						/>
						<LiveToggle
							live={live()}
							paused={!isNewest()}
							onToggle={() => setLive(!live())}
						/>
					</div>
				</div>
				<ActivePills search={search()} set={set} />
				<div
					class={cn("grid gap-5", facetsOpen() && "lg:grid-cols-[192px_1fr]")}
				>
					<Show when={facetsOpen()}>
						<FacetPanel
							search={search()}
							counts={summary.data?.facets}
							set={set}
						/>
					</Show>
					<div class="min-w-0">
						<Show
							when={summary.data}
							fallback={
								<div class="mb-3 h-[92px] rounded-xl border border-border bg-surface" />
							}
						>
							{(s) => <Histogram summary={s()} />}
						</Show>
						<RunLog
							query={page}
							total={summary.data?.total}
							offset={
								search().before && cursors().length === 0
									? undefined
									: cursors().length * PAGE_LIMIT
							}
							onClear={() => set(CLEARED)}
							onNewest={() => set({})}
							onNewer={() => {
								const prev = cursors();
								setCursors(prev.slice(0, -1));
								go({ before: prev.at(-1) });
							}}
							onOlder={(cursor) => {
								setCursors([...cursors(), search().before]);
								go({ before: cursor });
							}}
						/>
					</div>
				</div>
			</section>
		</div>
	);
}

function LiveToggle(props: {
	live: boolean;
	paused: boolean;
	onToggle: () => void;
}) {
	const on = () => props.live && !props.paused;
	return (
		<button
			type="button"
			aria-pressed={props.live}
			title={
				props.paused
					? "Live updates pause while you browse older runs"
					: undefined
			}
			onClick={() => props.onToggle()}
			class={cn(
				"inline-flex h-8 items-center gap-2 rounded-md px-2 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
				on()
					? "text-accent-text hover:bg-accent-subtle"
					: "text-faint hover:bg-surface-muted",
			)}
		>
			<span class="relative flex size-2">
				<Show when={on()}>
					<span class="absolute inset-0 animate-ping rounded-full bg-primary/60 motion-reduce:hidden" />
				</Show>
				<span
					class={cn(
						"relative size-2 rounded-full",
						on() ? "bg-primary" : "bg-border-strong",
					)}
				/>
			</span>
			{on() ? "Live" : "Paused"}
		</button>
	);
}

function toLocalInput(ms: number) {
	const d = new Date(ms - new Date(ms).getTimezoneOffset() * 60_000);
	return d.toISOString().slice(0, 16);
}

function rangeLabel(s: RunQuery) {
	if (s.from && s.to) {
		const fmt = (iso: string) =>
			new Date(iso).toLocaleString("en-GB", {
				day: "numeric",
				month: "short",
				hour: "2-digit",
				minute: "2-digit",
			});
		return `${fmt(s.from)} – ${fmt(s.to)}`;
	}
	return (
		PRESETS.find((p) => p.value === (s.range ?? DEFAULT_RANGE)) ?? PRESETS[3]
	).label;
}

function DateRangePicker(props: {
	search: RunQuery;
	onChange: (r: {
		range?: string | undefined;
		from?: string | undefined;
		to?: string | undefined;
	}) => void;
}) {
	const [open, setOpen] = createSignal(false);
	const now = () => Date.now();
	const [from, setFrom] = createSignal("");
	const [to, setTo] = createSignal("");
	const custom = () => !!(props.search.from && props.search.to);
	const current = () =>
		custom() ? "custom" : (props.search.range ?? DEFAULT_RANGE);
	const invalid = () =>
		!from() || !to() || Date.parse(from()) >= Date.parse(to());
	const onOpenChange = (next: boolean) => {
		if (next) {
			const w = resolveWindow(props.search);
			setFrom(toLocalInput(w.from));
			setTo(toLocalInput(w.to));
		}
		setOpen(next);
	};

	return (
		<Popover
			open={open()}
			onOpenChange={onOpenChange}
			placement="bottom-end"
			gutter={6}
		>
			<Popover.Trigger
				as={(p: JSX.HTMLAttributes<HTMLButtonElement>) => (
					<Button variant="outline" size="sm" {...p}>
						<Icon name="history" size={14} />
						{rangeLabel(props.search)}
						<Icon name="chevronDown" size={14} class="text-faint" />
					</Button>
				)}
			/>
			<Popover.Portal>
				<Popover.Content class="z-50 w-72 rounded-xl border border-border bg-surface p-1.5 shadow-xl outline-none data-[expanded]:animate-in data-[expanded]:fade-in-0 data-[expanded]:zoom-in-95">
					<ul aria-label="Time range">
						<For each={PRESETS}>
							{(p) => (
								<li>
									<button
										type="button"
										class={cn(
											"flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left text-sm hover:bg-surface-muted focus-visible:bg-surface-muted focus-visible:outline-none",
											current() === p.value
												? "font-medium text-accent-text"
												: "text-foreground",
										)}
										onClick={() => {
											props.onChange({
												range: p.value === DEFAULT_RANGE ? undefined : p.value,
												from: undefined,
												to: undefined,
											});
											setOpen(false);
										}}
									>
										{p.label}
										<Show when={current() === p.value}>
											<Icon name="check" size={14} />
										</Show>
									</button>
								</li>
							)}
						</For>
					</ul>
					<form
						class="mt-1.5 space-y-2 border-t border-border px-2.5 pt-2.5 pb-1.5"
						onSubmit={(e) => {
							e.preventDefault();
							if (invalid()) return;
							props.onChange({
								range: undefined,
								from: new Date(from()).toISOString(),
								to: new Date(to()).toISOString(),
							});
							setOpen(false);
						}}
					>
						<p
							class={cn(
								"text-xs font-semibold",
								custom() ? "text-accent-text" : "text-foreground",
							)}
						>
							Custom range
						</p>
						<label class="grid grid-cols-[40px_1fr] items-center gap-2 text-xs text-faint">
							From
							<input
								type="datetime-local"
								class="h-8 rounded-md border border-border bg-surface px-2 font-mono text-xs text-foreground focus:border-primary focus:ring-2 focus:ring-primary/10 focus:outline-none"
								min={toLocalInput(now() - RETENTION_DAYS * DAY)}
								max={toLocalInput(now())}
								value={from()}
								onInput={(e) => setFrom(e.currentTarget.value)}
							/>
						</label>
						<label class="grid grid-cols-[40px_1fr] items-center gap-2 text-xs text-faint">
							To
							<input
								type="datetime-local"
								class="h-8 rounded-md border border-border bg-surface px-2 font-mono text-xs text-foreground focus:border-primary focus:ring-2 focus:ring-primary/10 focus:outline-none"
								min={toLocalInput(now() - RETENTION_DAYS * DAY)}
								max={toLocalInput(now())}
								value={to()}
								onInput={(e) => setTo(e.currentTarget.value)}
							/>
						</label>
						<div class="flex items-center justify-between pt-1">
							<span class="text-2xs text-faint">Runs are kept for 30 days</span>
							<Button type="submit" size="sm" disabled={invalid()}>
								Apply
							</Button>
						</div>
					</form>
				</Popover.Content>
			</Popover.Portal>
		</Popover>
	);
}

function HealthDot(props: { health: Health }) {
	return (
		<span
			aria-hidden="true"
			class={cn("size-2 shrink-0 rounded-full", HEALTH_DOT[props.health])}
		/>
	);
}

function LastRuns(props: { targetId: string }) {
	const runs = () => lastFive(props.targetId).reverse();
	return (
		<span
			class="inline-flex items-end gap-[3px]"
			aria-label="Last 5 runs, oldest first"
		>
			<For each={runs()}>
				{(r) => (
					<span
						title={`${formatRelative(r.startedAt)} · ${OUTCOME_LABEL[outcomeOf(r)]} · ${r.fresh} new`}
						class={cn(
							"h-4 w-1.5 rounded-[2px]",
							outcomeOf(r) === "ok" && r.fresh === 0
								? "bg-border-strong"
								: OUTCOME_DOT[outcomeOf(r)],
						)}
					/>
				)}
			</For>
		</span>
	);
}

function HealthTable(props: {
	selected: string | undefined;
	onSelect: (id: string | undefined) => void;
}) {
	const [showHealthy, setShowHealthy] = createSignal(false);
	const sorted = [...TARGETS].sort(
		(a, b) =>
			HEALTH_ORDER[healthOf(a.id)] - HEALTH_ORDER[healthOf(b.id)] ||
			a.name.localeCompare(b.name),
	);
	const count = (h: Health) =>
		TARGETS.filter((t) => healthOf(t.id) === h).length;
	const visible = () =>
		sorted.filter(
			(t) =>
				showHealthy() || healthOf(t.id) !== "green" || t.id === props.selected,
		);

	return (
		<section>
			<div class="mb-3 flex items-baseline gap-4">
				<h2 class="text-base font-semibold text-foreground">Sources</h2>
				<p class="flex gap-3 text-xs text-muted">
					<For
						each={
							[
								["red", "failing"],
								["amber", "need a look"],
								["green", "healthy"],
							] as const
						}
					>
						{([h, label]) => (
							<span class="inline-flex items-center gap-1.5">
								<HealthDot health={h} />
								<span class="font-mono tabular-nums text-foreground">
									{count(h)}
								</span>{" "}
								{label}
							</span>
						)}
					</For>
				</p>
			</div>
			<div class="overflow-hidden rounded-xl border border-border bg-surface">
				<Table>
					<TableHeader>
						<TableRow class="hover:bg-transparent">
							<TableHead class="h-9 w-[38%]">Search or board</TableHead>
							<TableHead class="h-9">Status</TableHead>
							<TableHead class="h-9">Last 5</TableHead>
							<TableHead class="h-9">Last run</TableHead>
							<TableHead class="h-9 text-right">New</TableHead>
							<TableHead class="h-9 text-right">Elsewhere</TableHead>
							<TableHead class="h-9 text-right">Retries</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						<For each={visible()}>
							{(t) => {
								const last5 = lastFive(t.id);
								const sum = (pick: (r: Run) => number) =>
									last5.reduce((n, r) => n + pick(r), 0);
								const isSelected = () => props.selected === t.id;
								const toggle = () =>
									props.onSelect(isSelected() ? undefined : t.id);
								return (
									<TableRow
										tabIndex={0}
										aria-selected={isSelected()}
										class={cn(
											"cursor-pointer focus-visible:bg-surface-muted focus-visible:outline-none",
											isSelected() && "bg-accent-subtle hover:bg-accent-subtle",
										)}
										onClick={toggle}
										onKeyDown={(e) => {
											if (e.key === "Enter" || e.key === " ") {
												e.preventDefault();
												toggle();
											}
										}}
									>
										<TableCell class="py-2.5">
											<div class="flex items-center gap-2.5">
												<SourceBadge source={t.source} />
												<span class="truncate font-medium">{t.name}</span>
												<span class="text-xs text-faint">
													{t.kind === "search" ? "Search" : "Board"}
												</span>
											</div>
										</TableCell>
										<TableCell class="py-2.5">
											<span
												class={cn(
													"inline-flex items-center gap-2 text-xs",
													healthOf(t.id) === "red"
														? "font-medium text-destructive-strong"
														: "text-muted",
												)}
											>
												<HealthDot health={healthOf(t.id)} />
												{healthReason(t.id)}
											</span>
										</TableCell>
										<TableCell class="py-2.5">
											<LastRuns targetId={t.id} />
										</TableCell>
										<TableCell class="py-2.5 text-xs text-muted">
											{formatRelative(
												RUNS.find((r) => r.target.id === t.id)?.startedAt ?? "",
											)}
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num
												value={sum((r) => r.fresh)}
												emphasis="font-semibold text-accent-text"
											/>
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num value={sum((r) => r.elsewhere)} />
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num
												value={sum((r) => r.errors.length)}
												emphasis="font-medium text-destructive-strong"
											/>
										</TableCell>
									</TableRow>
								);
							}}
						</For>
					</TableBody>
				</Table>
				<button
					type="button"
					class="flex w-full items-center justify-center gap-1.5 border-t border-border py-2 text-xs font-medium text-muted hover:bg-surface-muted hover:text-foreground focus-visible:bg-surface-muted focus-visible:outline-none"
					onClick={() => setShowHealthy(!showHealthy())}
				>
					{showHealthy()
						? "Hide healthy sources"
						: `Show ${count("green")} healthy sources`}
					<Icon name={showHealthy() ? "chevronUp" : "chevronDown"} size={14} />
				</button>
			</div>
			<p class="mt-2 text-xs text-faint">
				Totals cover each source's last 5 runs. Select a row to filter the run
				log.
			</p>
		</section>
	);
}

function Num(props: { value: number; emphasis?: string }) {
	return (
		<span
			class={cn(
				"font-mono tabular-nums",
				props.value === 0 ? "text-faint" : props.emphasis,
			)}
		>
			{props.value === 0 ? "–" : props.value.toLocaleString()}
		</span>
	);
}

function FacetPanel(props: {
	search: ActivitySearch;
	counts: RunsSummary["facets"] | undefined;
	set: (p: Patch) => void;
}) {
	return (
		<aside
			id="run-facets"
			aria-label="Filter runs"
			class="space-y-5 lg:sticky lg:top-6 lg:self-start"
		>
			<For each={FACETS}>
				{(f) => {
					const picked = () => props.search[f.key] ?? [];
					const toggle = (value: string) => {
						const next = picked().includes(value)
							? picked().filter((v) => v !== value)
							: [...picked(), value];
						props.set({ [f.key]: next.length ? next : undefined });
					};
					return (
						<fieldset>
							<legend class="mb-1.5 flex w-full items-center justify-between text-xs font-semibold text-foreground">
								{f.label}
								<Show when={picked().length > 0}>
									<button
										type="button"
										class="rounded px-1 text-xs font-normal text-faint hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
										onClick={() => props.set({ [f.key]: undefined })}
									>
										Reset
									</button>
								</Show>
							</legend>
							<ul>
								<For each={f.values}>
									{(v) => {
										const n = () => props.counts?.[f.key][v.value] ?? 0;
										const on = () => picked().includes(v.value);
										return (
											<li
												class={cn(
													"group -mx-2 flex items-center gap-2 rounded-md px-2 py-1 text-data hover:bg-surface",
													n() === 0 && !on() && "opacity-50",
												)}
											>
												<input
													id={`facet-${f.key}-${v.value}`}
													type="checkbox"
													class="size-3.5 shrink-0 cursor-pointer accent-primary"
													checked={on()}
													onChange={() => toggle(v.value)}
												/>
												<label
													for={`facet-${f.key}-${v.value}`}
													class={cn(
														"flex min-w-0 flex-1 cursor-pointer items-center gap-2 truncate",
														on() ? "text-foreground" : "text-muted",
													)}
												>
													<Show when={f.key === "outcome"}>
														<span
															class={cn(
																"size-1.5 shrink-0 rounded-full",
																OUTCOME_DOT[v.value as Outcome],
															)}
														/>
													</Show>
													{v.label}
												</label>
												<button
													type="button"
													class="hidden rounded px-1 text-xs text-accent-text hover:underline focus-visible:inline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary group-hover:inline"
													onClick={() => props.set({ [f.key]: [v.value] })}
												>
													Only
												</button>
												<span class="font-mono text-xs tabular-nums text-faint group-hover:hidden">
													{n().toLocaleString()}
												</span>
											</li>
										);
									}}
								</For>
							</ul>
						</fieldset>
					);
				}}
			</For>
		</aside>
	);
}

function ActivePills(props: {
	search: ActivitySearch;
	set: (p: Patch) => void;
}) {
	const target = () => TARGETS.find((t) => t.id === props.search.target);
	const pills = () =>
		FACETS.flatMap((f) => {
			const picked = props.search[f.key];
			if (!picked) return [];
			const labels = picked.map(
				(p) => f.values.find((v) => v.value === p)?.label ?? p,
			);
			return [{ key: f.key, label: f.label, value: labels.join(", ") }];
		});
	return (
		<Show when={pills().length > 0 || target() || props.search.q}>
			<div class="mb-3 flex flex-wrap items-center gap-1.5">
				<Show when={target()}>
					{(t) => (
						<Pill
							label="Search or board"
							value={t().name}
							onRemove={() => props.set({ target: undefined })}
						/>
					)}
				</Show>
				<For each={pills()}>
					{(p) => (
						<Pill
							label={p.label}
							value={p.value}
							onRemove={() => props.set({ [p.key]: undefined })}
						/>
					)}
				</For>
				<button
					type="button"
					class="rounded px-1.5 py-0.5 text-xs text-faint hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					onClick={() => props.set(CLEARED)}
				>
					Clear all
				</button>
			</div>
		</Show>
	);
}

function Pill(props: { label: string; value: string; onRemove: () => void }) {
	return (
		<span class="inline-flex items-center overflow-hidden rounded-md border border-accent-border bg-accent-subtle text-xs">
			<span class="py-0.5 pr-1 pl-2 text-accent-text/80">{props.label} is</span>
			<span class="max-w-64 truncate py-0.5 pr-1 font-medium text-accent-text">
				{props.value}
			</span>
			<button
				type="button"
				aria-label={`Remove ${props.label} filter`}
				class="flex h-full items-center px-1.5 text-accent-text hover:bg-accent-border/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
				onClick={() => props.onRemove()}
			>
				<Icon name="x" size={12} />
			</button>
		</span>
	);
}

function Histogram(props: { summary: RunsSummary }) {
	const max = () =>
		Math.max(
			1,
			...props.summary.histogram.map((b) => b.ok + b.errored + b.failed),
		);
	const daily = () => props.summary.bucketMs >= DAY;
	const fmt = (t: number) =>
		new Date(t).toLocaleString(
			"en-GB",
			daily()
				? { day: "numeric", month: "short" }
				: { weekday: "short", hour: "2-digit", minute: "2-digit" },
		);
	const at = (i: number) => props.summary.histogram[i]?.start ?? 0;
	const n = () => props.summary.histogram.length;

	return (
		<figure class="mb-3 rounded-xl border border-border bg-surface px-4 pt-3 pb-2">
			<figcaption class="sr-only">Runs over time by outcome</figcaption>
			<div class="flex h-14 items-end gap-[3px]">
				<For each={props.summary.histogram}>
					{(b) => {
						const total = b.ok + b.errored + b.failed;
						return (
							<div
								class="flex min-w-0 flex-1 flex-col-reverse overflow-hidden rounded-[2px]"
								style={{
									height: `${Math.max(total ? 6 : 2, (total / max()) * 100)}%`,
								}}
								title={`${fmt(b.start)} · ${total} runs · ${b.failed} failed · ${b.errored} retried`}
							>
								<Show when={total} fallback={<span class="h-full bg-border" />}>
									<span class="bg-border-strong" style={{ flex: b.ok }} />
									<span
										class="bg-status-interview"
										style={{ flex: b.errored }}
									/>
									<span class="bg-destructive" style={{ flex: b.failed }} />
								</Show>
							</div>
						);
					}}
				</For>
			</div>
			<div class="mt-1.5 flex justify-between font-mono text-2xs text-faint">
				<span>{fmt(at(0))}</span>
				<span>{fmt(at(Math.floor(n() / 2)))}</span>
				<span>{fmt(at(n() - 1))}</span>
			</div>
		</figure>
	);
}

const STOP_LABEL = {
	last_page: "Reached the last page",
	page_cap: "Hit the page cap",
	failed: "Gave up after retries",
};

function duration(sec: number) {
	return sec < 60 ? `${sec}s` : `${Math.floor(sec / 60)}m ${sec % 60}s`;
}

function clock(iso: string) {
	return new Date(iso).toLocaleString("en-GB", {
		day: "2-digit",
		month: "short",
		hour: "2-digit",
		minute: "2-digit",
	});
}

function RunLog(props: {
	query: { isPending: boolean; isError: boolean; data: RunsPage | undefined };
	total: number | undefined;
	offset: number | undefined;
	onClear: () => void;
	onNewest: () => void;
	onNewer: () => void;
	onOlder: (cursor: string) => void;
}) {
	const [open, setOpen] = createSignal<string>();
	const toggle = (id: string) => setOpen(open() === id ? undefined : id);
	const items = () => props.query.data?.items ?? [];

	return (
		<div class="overflow-hidden rounded-xl border border-border bg-surface">
			<Show when={!props.query.isPending} fallback={<SkeletonList rows={10} />}>
				<Show
					when={items().length > 0}
					fallback={
						<div class="px-6 py-12 text-center">
							<p class="text-sm text-foreground">
								No runs match these filters in this time range.
							</p>
							<button
								type="button"
								class="mt-2 text-xs font-medium text-accent-text hover:underline"
								onClick={() => props.onClear()}
							>
								Clear filters
							</button>
						</div>
					}
				>
					<Table class="[&_td]:px-2.5 [&_th]:px-2.5">
						<TableHeader>
							<TableRow class="hover:bg-transparent">
								<TableHead class="h-9 w-8" />
								<TableHead class="h-9">Started</TableHead>
								<TableHead class="h-9">Outcome</TableHead>
								<TableHead class="h-9">Search or board</TableHead>
								<TableHead class="hidden h-9 text-right 2xl:table-cell">
									Pages
								</TableHead>
								<TableHead class="h-9 text-right">New</TableHead>
								<TableHead class="h-9 text-right">Elsewhere</TableHead>
								<TableHead class="hidden h-9 text-right 2xl:table-cell">
									Repeat
								</TableHead>
								<TableHead class="h-9 pr-5 text-right">Retries</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							<For each={items()}>
								{(r) => {
									const isOpen = () => open() === r.id;
									return (
										<>
											<TableRow
												tabIndex={0}
												aria-expanded={isOpen()}
												class={cn(
													"cursor-pointer focus-visible:bg-surface-muted focus-visible:outline-none",
													isOpen() && "border-b-0 bg-surface-muted",
												)}
												onClick={() => toggle(r.id)}
												onKeyDown={(e) => {
													if (e.key === "Enter" || e.key === " ") {
														e.preventDefault();
														toggle(r.id);
													}
												}}
											>
												<TableCell class="w-8 py-2 text-faint">
													<Icon
														name="chevronRight"
														size={14}
														class={cn(
															"transition-transform duration-150 motion-reduce:transition-none",
															isOpen() && "rotate-90",
														)}
													/>
												</TableCell>
												<TableCell
													class="py-2"
													title={formatRelative(r.startedAt)}
												>
													<span class="font-mono text-xs tabular-nums text-muted">
														{clock(r.startedAt)}
													</span>
													<Show when={r.trigger === "manual"}>
														<span
															title="Run manually"
															class="ml-1.5 inline-flex align-[-2px] text-faint"
														>
															<Icon name="user" size={12} />
															<span class="sr-only">Run manually</span>
														</span>
													</Show>
												</TableCell>
												<TableCell class="py-2">
													<span
														class={cn(
															"inline-flex items-center gap-2 text-xs font-medium",
															outcomeOf(r) === "failed"
																? "text-destructive-strong"
																: "text-muted",
															outcomeOf(r) === "running" && "text-accent-text",
														)}
													>
														<span
															class={cn(
																"size-2 rounded-full",
																OUTCOME_DOT[outcomeOf(r)],
																outcomeOf(r) === "running" &&
																	"animate-pulse motion-reduce:animate-none",
															)}
														/>
														{OUTCOME_LABEL[outcomeOf(r)]}
														<Show when={r.stopReason === "page_cap"}>
															<span
																class="font-normal text-status-interview"
																title="Stopped at the page cap with more pages left"
															>
																· capped
															</span>
														</Show>
													</span>
												</TableCell>
												<TableCell class="max-w-48 py-2">
													<div class="flex items-center gap-2">
														<SourceBadge source={r.target.source} />
														<span class="truncate">{r.target.name}</span>
													</div>
												</TableCell>
												<TableCell class="hidden py-2 text-right 2xl:table-cell">
													<Num value={r.pages} />
												</TableCell>
												<TableCell class="py-2 text-right">
													<Num
														value={r.fresh}
														emphasis="font-semibold text-accent-text"
													/>
												</TableCell>
												<TableCell class="py-2 text-right">
													<Num value={r.elsewhere} />
												</TableCell>
												<TableCell class="hidden py-2 text-right 2xl:table-cell">
													<Num value={r.repeat} />
												</TableCell>
												<TableCell class="py-2 pr-5 text-right">
													<Num
														value={r.errors.length}
														emphasis="font-medium text-destructive-strong"
													/>
												</TableCell>
											</TableRow>
											<Show when={isOpen()}>
												<tr class="border-b border-border bg-surface-muted">
													<td colSpan={9} class="pt-0 pb-3">
														<RunDetail run={r} />
													</td>
												</tr>
											</Show>
										</>
									);
								}}
							</For>
						</TableBody>
					</Table>
					<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
						<span class="mr-auto text-xs text-faint" aria-live="polite">
							<Show
								when={props.offset !== undefined}
								fallback={`${items().length} older runs`}
							>
								<span class="font-mono tabular-nums">
									{((props.offset ?? 0) + 1).toLocaleString()}–
									{((props.offset ?? 0) + items().length).toLocaleString()}
								</span>
							</Show>
							<Show when={props.total !== undefined}>
								{" "}
								of{" "}
								<span class="font-mono tabular-nums">
									{props.total?.toLocaleString()}
								</span>{" "}
								runs
							</Show>
						</span>
						<Show when={props.offset !== 0}>
							<Button
								variant="ghost"
								size="sm"
								onClick={() => props.onNewest()}
							>
								Newest
							</Button>
							<Button
								variant="outline"
								size="sm"
								onClick={() => props.onNewer()}
							>
								<Icon name="chevronLeft" size={14} />
								Newer
							</Button>
						</Show>
						<Button
							variant="outline"
							size="sm"
							disabled={!props.query.data?.nextCursor}
							onClick={() => {
								const c = props.query.data?.nextCursor;
								if (c) props.onOlder(c);
							}}
						>
							Older
							<Icon name="chevronRight" size={14} />
						</Button>
					</div>
				</Show>
			</Show>
		</div>
	);
}

function RunDetail(props: { run: Run }) {
	const r = () => props.run;
	const facts = () => [
		[
			"Stopped",
			r().stopReason
				? STOP_LABEL[r().stopReason as keyof typeof STOP_LABEL]
				: "Still running",
		],
		[
			"Listings",
			`${r().listed} across ${r().pages} ${r().pages === 1 ? "page" : "pages"}, ${r().repeat} repeats`,
		],
		["Trigger", r().trigger === "manual" ? "Run by you" : "Scheduled"],
		["Duration", r().status === "running" ? "–" : duration(r().durationSec)],
		r().target.kind === "board"
			? ["Removed", `${r().removed} since last poll`]
			: ["Rate-limited", r().deferred ? `Waited ${r().deferred}×` : "No waits"],
	];
	return (
		<div class="ml-8 mr-2 grid gap-x-8 gap-y-3 md:grid-cols-[240px_1fr]">
			<dl class="grid grid-cols-[auto_1fr] content-start gap-x-4 gap-y-1 text-xs">
				<For each={facts()}>
					{([k, v]) => (
						<>
							<dt class="text-faint">{k}</dt>
							<dd
								class={cn(
									"text-foreground",
									r().stopReason === "page_cap" &&
										k === "Stopped" &&
										"font-medium text-status-interview",
								)}
							>
								{v}
							</dd>
						</>
					)}
				</For>
			</dl>
			<Show
				when={r().errors.length > 0}
				fallback={
					<p class="text-xs text-faint">
						Every page loaded on the first attempt.
					</p>
				}
			>
				<ol class="overflow-hidden rounded-lg border border-border bg-surface font-mono text-xs">
					<For each={r().errors}>
						{(e) => (
							<li class="flex gap-4 border-b border-border px-3 py-1.5 last:border-0">
								<span class="shrink-0 tabular-nums text-faint">
									{new Date(e.at).toLocaleTimeString("en-GB")}
								</span>
								<span class="shrink-0 text-muted">
									p{e.page} · try {e.attempt}
								</span>
								<span class="min-w-0 break-words text-destructive-strong">
									{e.message}
								</span>
							</li>
						)}
					</For>
				</ol>
			</Show>
		</div>
	);
}
