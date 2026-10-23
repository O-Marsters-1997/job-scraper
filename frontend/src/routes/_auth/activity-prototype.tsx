import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { Pager } from "@/components/Pager";
import { SearchField } from "@/components/SearchField";
import { SourceBadge } from "@/components/SourceBadge";
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

const TARGETS: (Target & {
	flavour?: "failing" | "flaky" | "stale" | "capped";
})[] = [
	{
		id: "t1",
		kind: "search",
		source: "LinkedIn",
		name: "Senior Go Engineer · London",
		flavour: "capped",
	},
	{
		id: "t2",
		kind: "search",
		source: "LinkedIn",
		name: "Platform Engineer · Remote UK",
	},
	{
		id: "t3",
		kind: "search",
		source: "Indeed",
		name: "Backend Developer · Manchester",
		flavour: "failing",
	},
	{
		id: "t4",
		kind: "search",
		source: "hiring.cafe",
		name: "Software Engineer",
		flavour: "stale",
	},
	{ id: "t5", kind: "search", source: "Wellfound", name: "Founding Engineer" },
	{ id: "b1", kind: "board", source: "Greenhouse", name: "Monzo" },
	{
		id: "b2",
		kind: "board",
		source: "Greenhouse",
		name: "Cleo",
		flavour: "flaky",
	},
	{ id: "b3", kind: "board", source: "Lever", name: "Octopus Energy" },
	{
		id: "b4",
		kind: "board",
		source: "Ashby",
		name: "Synthesia",
		flavour: "failing",
	},
	{
		id: "b5",
		kind: "board",
		source: "Workable",
		name: "Bloom & Wild",
		flavour: "stale",
	},
];

const ERROR_MESSAGES = [
	"GET page: 503 Service Unavailable",
	"parse listing cards: no .job-card nodes found",
	"context deadline exceeded after 30s",
	"GET board: 404 Not Found (token may have changed)",
];

function seeded(seed: number) {
	let s = seed;
	return () => {
		s = (s * 1664525 + 1013904223) % 4294967296;
		return s / 4294967296;
	};
}

function makeRuns(): Run[] {
	const rand = seeded(42);
	const int = (lo: number, hi: number) =>
		lo + Math.floor(rand() * (hi - lo + 1));
	const now = Date.now();
	const runs: Run[] = [];
	for (const t of TARGETS) {
		for (let i = 0; i < 12; i++) {
			const start = now - (i * 6 + rand() * 2) * 3600_000 - int(1, 50) * 60_000;
			const failing = t.flavour === "failing" && i < 3;
			const flaky = t.flavour === "flaky" || rand() < 0.08;
			const running = i === 0 && t.id === "t2";
			const pages =
				t.kind === "board"
					? int(1, 2)
					: t.flavour === "capped"
						? 10
						: int(2, 9);
			const at = (offsetSec: number) =>
				new Date(start + offsetSec * 1000).toISOString();
			const errors: RunError[] = [];
			if (failing) {
				for (let a = 1; a <= 3; a++)
					errors.push({
						page: 1,
						attempt: a,
						message: ERROR_MESSAGES[t.kind === "board" ? 3 : 1] ?? "",
						at: at(a * 20),
					});
			} else if (flaky && !running) {
				errors.push({
					page: int(1, pages),
					attempt: 1,
					message: ERROR_MESSAGES[int(0, 2)] ?? "",
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
				id: `${t.id}-${i}`,
				target: t,
				startedAt: new Date(start).toISOString(),
				durationSec: running
					? 0
					: t.kind === "board"
						? int(2, 15)
						: pages * int(8, 25),
				status: running ? "running" : failing ? "failed" : "succeeded",
				trigger: rand() < 0.15 ? "manual" : "scheduled",
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
		}
	}
	return runs.sort((a, b) => b.startedAt.localeCompare(a.startedAt));
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
	running: "bg-primary animate-pulse motion-reduce:animate-none",
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

const PAGE_SIZE = 20;
const BUCKET_HOURS = 6;
const BUCKETS = 12;

interface ActivitySearch {
	q?: string | undefined;
	outcome?: string[] | undefined;
	kind?: string[] | undefined;
	source?: string[] | undefined;
	trigger?: string[] | undefined;
	target?: string | undefined;
	page?: number | undefined;
}

type Patch = Partial<ActivitySearch>;

const list = (v: unknown) => {
	const arr = (Array.isArray(v) ? v : [v]).filter(
		(x): x is string => typeof x === "string" && x !== "",
	);
	return arr.length ? arr : undefined;
};

export const Route = createFileRoute("/_auth/activity-prototype")({
	validateSearch: (s: Record<string, unknown>): ActivitySearch => ({
		q: typeof s.q === "string" && s.q ? s.q : undefined,
		outcome: list(s.outcome),
		kind: list(s.kind),
		source: list(s.source),
		trigger: list(s.trigger),
		target: typeof s.target === "string" ? s.target : undefined,
		page: Number(s.page) > 1 ? Number(s.page) : undefined,
	}),
	component: ActivityPage,
});

function matches(r: Run, s: ActivitySearch, except?: FacetKey) {
	if (s.target && r.target.id !== s.target) return false;
	if (s.q) {
		const q = s.q.toLowerCase();
		const hay = [
			r.target.name,
			r.target.source,
			...r.errors.map((e) => e.message),
		]
			.join(" ")
			.toLowerCase();
		if (!hay.includes(q)) return false;
	}
	return FACETS.every((f) => {
		const picked = s[f.key];
		return f.key === except || !picked || picked.includes(f.of(r));
	});
}

function ActivityPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/activity-prototype" });
	const set = (patch: Patch) =>
		navigate({
			search: (prev) => ({ ...prev, page: undefined, ...patch }),
			replace: true,
		});
	const runs = createMemo(() => RUNS.filter((r) => matches(r, search())));
	let logEl: HTMLElement | undefined;

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Activity"
				subtitle="Scrape runs for your searches and tracked boards, last 30 days"
			/>
			<HealthTable
				selected={search().target}
				onSelect={(id) => {
					set({ target: id });
					if (id) logEl?.scrollIntoView({ behavior: "smooth", block: "start" });
				}}
			/>
			<section ref={logEl} class="mt-8 scroll-mt-6">
				<div class="mb-3 flex items-baseline justify-between gap-4">
					<h2 class="text-base font-semibold text-foreground">Run log</h2>
					<LiveToggle />
				</div>
				<div class="grid gap-5 lg:grid-cols-[192px_1fr]">
					<FacetPanel search={search()} set={set} />
					<div class="min-w-0">
						<Toolbar search={search()} set={set} total={runs().length} />
						<Histogram runs={runs()} />
						<RunLog
							runs={runs()}
							page={search().page ?? 1}
							onPage={(page) => set({ page })}
							onClear={() => set(CLEARED)}
						/>
					</div>
				</div>
			</section>
		</div>
	);
}

const CLEARED: Patch = {
	q: undefined,
	outcome: undefined,
	kind: undefined,
	source: undefined,
	trigger: undefined,
	target: undefined,
};

function LiveToggle() {
	const [live, setLive] = createSignal(true);
	return (
		<button
			type="button"
			aria-pressed={live()}
			onClick={() => setLive(!live())}
			class={cn(
				"inline-flex items-center gap-2 rounded-md px-2 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
				live()
					? "text-accent-text hover:bg-accent-subtle"
					: "text-faint hover:bg-surface-muted",
			)}
		>
			<span class="relative flex size-2">
				<Show when={live()}>
					<span class="absolute inset-0 animate-ping rounded-full bg-primary/60 motion-reduce:hidden" />
				</Show>
				<span
					class={cn(
						"relative size-2 rounded-full",
						live() ? "bg-primary" : "bg-border-strong",
					)}
				/>
			</span>
			{live() ? "Live" : "Paused"}
		</button>
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
	const targets = [...TARGETS].sort(
		(a, b) =>
			HEALTH_ORDER[healthOf(a.id)] - HEALTH_ORDER[healthOf(b.id)] ||
			a.name.localeCompare(b.name),
	);
	const count = (h: Health) =>
		TARGETS.filter((t) => healthOf(t.id) === h).length;

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
						<For each={targets}>
							{(t) => {
								const last5 = lastFive(t.id);
								const sum = (pick: (r: Run) => number) =>
									last5.reduce((n, r) => n + pick(r), 0);
								const isSelected = () => props.selected === t.id;
								return (
									<TableRow
										tabIndex={0}
										aria-selected={isSelected()}
										class={cn(
											"cursor-pointer focus-visible:bg-surface-muted focus-visible:outline-none",
											isSelected() && "bg-accent-subtle hover:bg-accent-subtle",
										)}
										onClick={() =>
											props.onSelect(isSelected() ? undefined : t.id)
										}
										onKeyDown={(e) => {
											if (e.key === "Enter" || e.key === " ") {
												e.preventDefault();
												props.onSelect(isSelected() ? undefined : t.id);
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
												emphasis="text-accent-text font-semibold"
											/>
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num value={sum((r) => r.elsewhere)} />
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num
												value={sum((r) => r.errors.length)}
												emphasis="text-destructive-strong font-medium"
											/>
										</TableCell>
									</TableRow>
								);
							}}
						</For>
					</TableBody>
				</Table>
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
	set: (p: Patch) => void;
}) {
	return (
		<aside
			aria-label="Filter runs"
			class="space-y-5 lg:sticky lg:top-6 lg:self-start"
		>
			<For each={FACETS}>
				{(f) => {
					const picked = () => props.search[f.key] ?? [];
					const countOf = (value: string) =>
						RUNS.filter(
							(r) => matches(r, props.search, f.key) && f.of(r) === value,
						).length;
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
										const n = () => countOf(v.value);
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
													{n()}
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

function Toolbar(props: {
	search: ActivitySearch;
	set: (p: Patch) => void;
	total: number;
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
	const anyFilter = () => pills().length > 0 || !!target() || !!props.search.q;

	return (
		<div class="mb-3 space-y-2">
			<div class="flex items-center gap-3">
				<SearchField
					id="run-search"
					label="Search runs"
					placeholder="Search names and error messages"
					value={props.search.q ?? ""}
					onInput={(q) => props.set({ q: q || undefined })}
					class="sm:max-w-80"
				/>
				<span class="ml-auto text-xs text-faint">
					<span class="font-mono tabular-nums text-foreground">
						{props.total}
					</span>{" "}
					runs
				</span>
			</div>
			<Show when={anyFilter()}>
				<div class="flex flex-wrap items-center gap-1.5">
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
		</div>
	);
}

function Pill(props: { label: string; value: string; onRemove: () => void }) {
	return (
		<span class="inline-flex items-center overflow-hidden rounded-md border border-accent-border bg-accent-subtle text-xs">
			<span class="py-0.5 pr-1 pl-2 text-accent-text/80">{props.label} is</span>
			<span class="py-0.5 pr-1 font-medium text-accent-text">
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

function Histogram(props: { runs: Run[] }) {
	const end =
		Math.ceil(Date.now() / (BUCKET_HOURS * 3600_000)) * BUCKET_HOURS * 3600_000;
	const buckets = createMemo(() => {
		const out = Array.from({ length: BUCKETS }, (_, i) => ({
			start: end - (BUCKETS - i) * BUCKET_HOURS * 3600_000,
			ok: 0,
			errored: 0,
			failed: 0,
		}));
		for (const r of props.runs) {
			const idx =
				BUCKETS -
				1 -
				Math.floor((end - Date.parse(r.startedAt)) / (BUCKET_HOURS * 3600_000));
			const b = out[idx];
			const o = outcomeOf(r);
			if (!b) continue;
			if (o === "failed") b.failed++;
			else if (o === "errored") b.errored++;
			else b.ok++;
		}
		return out;
	});
	const max = () =>
		Math.max(1, ...buckets().map((b) => b.ok + b.errored + b.failed));
	const fmt = (t: number) =>
		new Date(t).toLocaleString("en-GB", {
			weekday: "short",
			hour: "2-digit",
			minute: "2-digit",
		});

	return (
		<figure class="mb-3 rounded-xl border border-border bg-surface px-4 pt-3 pb-2">
			<figcaption class="sr-only">
				Runs per {BUCKET_HOURS} hours by outcome
			</figcaption>
			<div class="flex h-14 items-end gap-1">
				<For each={buckets()}>
					{(b) => {
						const total = b.ok + b.errored + b.failed;
						return (
							<div
								class="flex flex-1 flex-col-reverse overflow-hidden rounded-[3px]"
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
				<span>{fmt(buckets()[0]?.start ?? end)}</span>
				<span>{fmt(buckets()[BUCKETS / 2]?.start ?? end)}</span>
				<span>Now</span>
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
	runs: Run[];
	page: number;
	onPage: (p: number) => void;
	onClear: () => void;
}) {
	const [open, setOpen] = createSignal<string>();
	const slice = () =>
		props.runs.slice((props.page - 1) * PAGE_SIZE, props.page * PAGE_SIZE);
	const toggle = (id: string) => setOpen(open() === id ? undefined : id);

	return (
		<div class="overflow-hidden rounded-xl border border-border bg-surface">
			<Show
				when={props.runs.length > 0}
				fallback={
					<div class="px-6 py-12 text-center">
						<p class="text-sm text-foreground">No runs match these filters.</p>
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
							<TableHead class="h-9 w-8 px-0" />
							<TableHead class="h-9 pl-0">Started</TableHead>
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
						<For each={slice()}>
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
											<TableCell class="w-8 py-2 pr-0 pl-3 text-faint">
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
												class="py-2 pl-0"
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
													emphasis="text-accent-text font-semibold"
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
													emphasis="text-destructive-strong font-medium"
												/>
											</TableCell>
										</TableRow>
										<Show when={isOpen()}>
											<tr class="border-b border-border bg-surface-muted">
												<td colSpan={9} class="px-0 pt-0 pb-3">
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
				<Pager
					total={props.runs.length}
					page={props.page}
					pageSize={PAGE_SIZE}
					noun="runs"
					onPage={props.onPage}
				/>
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
		<div class="ml-10 mr-4 grid gap-x-8 gap-y-3 md:grid-cols-[220px_1fr]">
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
