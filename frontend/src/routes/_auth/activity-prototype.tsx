import { createFileRoute, useNavigate } from "@tanstack/solid-router";
import { createMemo, createSignal, For, type JSX, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { Pager } from "@/components/Pager";
import { PrototypeSwitcher } from "@/components/PrototypeSwitcher";
import { SourceBadge } from "@/components/SourceBadge";
import { ToggleChip } from "@/components/ToggleChip";
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
type StatusFilter = "succeeded" | "failed" | "errors";

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
			const startedAt = new Date(
				now - (i * 6 + rand() * 2) * 3600_000 - int(1, 50) * 60_000,
			);
			const failing = t.flavour === "failing" && i < 3;
			const flaky = t.flavour === "flaky" || rand() < 0.08;
			const running = i === 0 && t.id === "t2";
			const errors: RunError[] = [];
			const pages =
				t.kind === "board"
					? int(1, 2)
					: t.flavour === "capped"
						? 10
						: int(2, 9);
			if (failing) {
				for (let a = 1; a <= 3; a++)
					errors.push({
						page: 1,
						attempt: a,
						message: ERROR_MESSAGES[t.kind === "board" ? 3 : 1] ?? "",
					});
			} else if (flaky && !running) {
				errors.push({
					page: int(1, pages),
					attempt: 1,
					message: ERROR_MESSAGES[int(0, 2)] ?? "",
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
				startedAt: startedAt.toISOString(),
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
		case "amber":
			return last5.every((r) => r.fresh === 0)
				? "No new listings in 5 runs"
				: `${last5.reduce((n, r) => n + r.errors.length, 0)} errored attempts in 5 runs`;
		default:
			return "Healthy";
	}
}

const HEALTH_ORDER: Record<Health, number> = { red: 0, amber: 1, green: 2 };
const HEALTH_DOT: Record<Health, string> = {
	red: "bg-destructive",
	amber: "bg-status-interview",
	green: "bg-status-offer",
};
const SOURCES = [...new Set(TARGETS.map((t) => t.source))];
const PAGE_SIZE = 15;

interface ActivitySearch {
	variant?: string | undefined;
	source?: string | undefined;
	kind?: Kind | undefined;
	status?: StatusFilter | undefined;
	target?: string | undefined;
	page?: number | undefined;
}

export const Route = createFileRoute("/_auth/activity-prototype")({
	validateSearch: (s: Record<string, unknown>): ActivitySearch => ({
		variant: typeof s.variant === "string" ? s.variant : undefined,
		source: typeof s.source === "string" ? s.source : undefined,
		kind: s.kind === "search" || s.kind === "board" ? s.kind : undefined,
		status:
			s.status === "succeeded" || s.status === "failed" || s.status === "errors"
				? s.status
				: undefined,
		target: typeof s.target === "string" ? s.target : undefined,
		page: Number(s.page) > 1 ? Number(s.page) : undefined,
	}),
	component: ActivityPrototype,
});

const VARIANTS = [
	{ key: "A", name: "Health table + run log" },
	{ key: "B", name: "Source list + detail" },
	{ key: "C", name: "Problems first + day feed" },
];

type Patch = Partial<ActivitySearch>;

function ActivityPrototype() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/activity-prototype" });
	const set = (patch: Patch) =>
		navigate({
			search: (prev) => ({ ...prev, page: undefined, ...patch }),
			replace: true,
		});

	const filtered = createMemo(() =>
		RUNS.filter(
			(r) =>
				(!search().source || r.target.source === search().source) &&
				(!search().kind || r.target.kind === search().kind) &&
				(!search().target || r.target.id === search().target) &&
				(search().status !== "succeeded" || r.status === "succeeded") &&
				(search().status !== "failed" || r.status === "failed") &&
				(search().status !== "errors" || r.errors.length > 0),
		),
	);
	const variant = () => search().variant ?? "A";
	const props = { search, set, filtered };

	return (
		<div class="px-7 py-6 pb-24">
			<PageHeading
				title="Activity"
				subtitle="Scrape runs for your searches and tracked boards over the last 30 days"
			/>
			<Show when={variant() === "A"}>
				<VariantA {...props} />
			</Show>
			<Show when={variant() === "B"}>
				<VariantB {...props} />
			</Show>
			<Show when={variant() === "C"}>
				<VariantC {...props} />
			</Show>
			<PrototypeSwitcher
				variants={VARIANTS}
				current={variant()}
				onChange={(key) =>
					navigate({
						search: (prev) => ({ ...prev, variant: key }),
						replace: true,
					})
				}
			/>
		</div>
	);
}

interface VariantProps {
	search: () => ActivitySearch;
	set: (patch: Patch) => void;
	filtered: () => Run[];
}

function targetsSorted(filter: (t: Target) => boolean = () => true) {
	return TARGETS.filter(filter).sort(
		(a, b) =>
			HEALTH_ORDER[healthOf(a.id)] - HEALTH_ORDER[healthOf(b.id)] ||
			a.name.localeCompare(b.name),
	);
}

function HealthDot(props: { health: Health; class?: string }) {
	return (
		<span
			role="img"
			aria-label={props.health}
			class={cn(
				"inline-block size-2.5 shrink-0 rounded-full",
				HEALTH_DOT[props.health],
				props.class,
			)}
		/>
	);
}

function LastRunsStrip(props: { targetId: string }) {
	const runs = () => lastFive(props.targetId).reverse();
	return (
		<span class="inline-flex gap-0.5">
			<For each={runs()}>
				{(r) => (
					<span
						title={`${formatRelative(r.startedAt)}: ${r.status}, ${r.fresh} new, ${r.errors.length} errors`}
						class={cn(
							"h-4 w-1.5 rounded-sm",
							r.status === "failed"
								? "bg-destructive"
								: r.errors.length > 0
									? "bg-status-interview"
									: r.fresh === 0
										? "bg-border-strong"
										: "bg-status-offer",
						)}
					/>
				)}
			</For>
		</span>
	);
}

function YieldBar(props: { run: Run; class?: string }) {
	const pct = (n: number) =>
		`${props.run.listed ? (n / props.run.listed) * 100 : 0}%`;
	return (
		<div
			class={cn(
				"flex h-1.5 w-28 shrink-0 overflow-hidden rounded-full bg-surface-muted",
				props.class,
			)}
			title={`${props.run.fresh} new · ${props.run.elsewhere} found elsewhere · ${props.run.repeat} repeats`}
		>
			<span class="bg-primary" style={{ width: pct(props.run.fresh) }} />
			<span
				class="bg-accent-border"
				style={{ width: pct(props.run.elsewhere) }}
			/>
			<span class="bg-border-strong" style={{ width: pct(props.run.repeat) }} />
		</div>
	);
}

function RunStatusText(props: { run: Run }) {
	return (
		<span
			class={cn(
				"text-xs font-medium",
				props.run.status === "failed" && "text-destructive-strong",
				props.run.status === "running" && "text-accent-text",
				props.run.status === "succeeded" && "text-muted",
			)}
		>
			<Show
				when={props.run.status === "running"}
				fallback={props.run.status === "failed" ? "Failed" : "Succeeded"}
			>
				Running…
			</Show>
		</span>
	);
}

const STOP_LABEL = {
	last_page: "reached last page",
	page_cap: "hit page cap",
	failed: "gave up after retries",
};

function duration(sec: number) {
	return sec < 60 ? `${sec}s` : `${Math.floor(sec / 60)}m ${sec % 60}s`;
}

function RunDetail(props: { run: Run }) {
	const r = () => props.run;
	return (
		<div class="grid gap-4 bg-surface-muted px-4 py-3 text-xs text-muted md:grid-cols-[260px_1fr]">
			<dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
				<dt class="text-faint">Trigger</dt>
				<dd>{r().trigger}</dd>
				<dt class="text-faint">Stopped</dt>
				<dd
					class={cn(
						r().stopReason === "page_cap" &&
							"font-medium text-status-interview",
					)}
				>
					{r().stopReason
						? STOP_LABEL[r().stopReason as keyof typeof STOP_LABEL]
						: "still running"}
				</dd>
				<dt class="text-faint">Duration</dt>
				<dd>{r().status === "running" ? "—" : duration(r().durationSec)}</dd>
				<Show when={r().target.kind === "search"}>
					<dt class="text-faint">Rate-limited</dt>
					<dd>{r().deferred} waits</dd>
				</Show>
				<Show when={r().target.kind === "board"}>
					<dt class="text-faint">Removed</dt>
					<dd>{r().removed} since last poll</dd>
				</Show>
			</dl>
			<Show
				when={r().errors.length > 0}
				fallback={<p class="text-faint">No errors.</p>}
			>
				<ul class="space-y-1 font-mono">
					<For each={r().errors}>
						{(e) => (
							<li>
								<span class="text-faint">
									page {e.page} · attempt {e.attempt}
								</span>{" "}
								<span class="text-destructive-strong">{e.message}</span>
							</li>
						)}
					</For>
				</ul>
			</Show>
		</div>
	);
}

function Filters(props: VariantProps & { hideTarget?: boolean }) {
	const target = () => TARGETS.find((t) => t.id === props.search().target);
	return (
		<div class="mb-3 flex flex-wrap items-center gap-2">
			<ToggleChip
				active={!props.search().kind}
				onClick={() => props.set({ kind: undefined })}
			>
				All
			</ToggleChip>
			<ToggleChip
				active={props.search().kind === "search"}
				onClick={() => props.set({ kind: "search" })}
			>
				Searches
			</ToggleChip>
			<ToggleChip
				active={props.search().kind === "board"}
				onClick={() => props.set({ kind: "board" })}
			>
				Boards
			</ToggleChip>
			<span class="mx-1 h-5 w-px bg-border" />
			<For each={SOURCES}>
				{(s) => (
					<ToggleChip
						active={props.search().source === s}
						onClick={() =>
							props.set({ source: props.search().source === s ? undefined : s })
						}
					>
						{s}
					</ToggleChip>
				)}
			</For>
			<span class="mx-1 h-5 w-px bg-border" />
			<For
				each={
					[
						["failed", "Failed"],
						["errors", "Has errors"],
						["succeeded", "Succeeded"],
					] as const
				}
			>
				{([value, label]) => (
					<ToggleChip
						active={props.search().status === value}
						onClick={() =>
							props.set({
								status: props.search().status === value ? undefined : value,
							})
						}
					>
						{label}
					</ToggleChip>
				)}
			</For>
			<Show when={!props.hideTarget && target()}>
				{(t) => (
					<ToggleChip
						active
						solid
						onClick={() => props.set({ target: undefined })}
					>
						{t().name} ✕
					</ToggleChip>
				)}
			</Show>
		</div>
	);
}

function Num(props: { value: number; class?: string; zero?: string }) {
	return (
		<span
			class={cn(
				"font-mono tabular-nums",
				props.value === 0 ? "text-faint" : props.class,
			)}
		>
			{props.value === 0 && props.zero ? props.zero : props.value}
		</span>
	);
}

function Paged(props: {
	runs: Run[];
	page: number;
	pageSize?: number;
	set: (p: Patch) => void;
	children: (runs: Run[]) => JSX.Element;
}) {
	const size = () => props.pageSize ?? PAGE_SIZE;
	const slice = () =>
		props.runs.slice((props.page - 1) * size(), props.page * size());
	return (
		<>
			{props.children(slice())}
			<Pager
				total={props.runs.length}
				page={props.page}
				pageSize={size()}
				noun="runs"
				onPage={(page) => props.set({ page })}
			/>
		</>
	);
}

function VariantA(props: VariantProps) {
	const [open, setOpen] = createSignal<string>();
	return (
		<>
			<section class="mb-6 overflow-hidden rounded-xl border border-border bg-surface">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Search / Board</TableHead>
							<TableHead>Health</TableHead>
							<TableHead>Last 5</TableHead>
							<TableHead>Last run</TableHead>
							<TableHead class="text-right">New (5 runs)</TableHead>
							<TableHead class="text-right">Elsewhere</TableHead>
							<TableHead class="text-right">Errors</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						<For each={targetsSorted()}>
							{(t) => {
								const last5 = lastFive(t.id);
								const sum = (k: "fresh" | "elsewhere") =>
									last5.reduce((n, r) => n + r[k], 0);
								return (
									<TableRow
										class={cn(
											"cursor-pointer",
											props.search().target === t.id && "bg-accent-subtle",
										)}
										onClick={() =>
											props.set({
												target:
													props.search().target === t.id ? undefined : t.id,
											})
										}
									>
										<TableCell class="py-2.5">
											<div class="flex items-center gap-2">
												<SourceBadge source={t.source} />
												<span class="font-medium">{t.name}</span>
												<span class="text-xs text-faint">{t.kind}</span>
											</div>
										</TableCell>
										<TableCell class="py-2.5">
											<span class="inline-flex items-center gap-2 text-xs text-muted">
												<HealthDot health={healthOf(t.id)} />
												{healthReason(t.id)}
											</span>
										</TableCell>
										<TableCell class="py-2.5">
											<LastRunsStrip targetId={t.id} />
										</TableCell>
										<TableCell class="py-2.5 text-xs text-muted">
											{formatRelative(
												RUNS.find((r) => r.target.id === t.id)?.startedAt ?? "",
											)}
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num value={sum("fresh")} />
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num value={sum("elsewhere")} />
										</TableCell>
										<TableCell class="py-2.5 text-right">
											<Num
												value={last5.reduce((n, r) => n + r.errors.length, 0)}
												class="text-destructive-strong"
											/>
										</TableCell>
									</TableRow>
								);
							}}
						</For>
					</TableBody>
				</Table>
			</section>

			<h2 class="mb-2 text-sm font-semibold text-foreground">Run log</h2>
			<Filters {...props} />
			<section class="overflow-hidden rounded-xl border border-border bg-surface">
				<Paged
					runs={props.filtered()}
					page={props.search().page ?? 1}
					set={props.set}
				>
					{(runs) => (
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Started</TableHead>
									<TableHead>Search / Board</TableHead>
									<TableHead>Status</TableHead>
									<TableHead class="text-right">Pages</TableHead>
									<TableHead class="text-right">Listed</TableHead>
									<TableHead class="text-right">New</TableHead>
									<TableHead class="text-right">Elsewhere</TableHead>
									<TableHead class="text-right">Repeat</TableHead>
									<TableHead class="text-right">Errors</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								<For each={runs}>
									{(r) => (
										<>
											<TableRow
												class="cursor-pointer"
												onClick={() =>
													setOpen(open() === r.id ? undefined : r.id)
												}
											>
												<TableCell class="py-2.5 text-xs text-muted">
													{formatRelative(r.startedAt)}
													<Show when={r.trigger === "manual"}>
														<span class="ml-1.5 text-faint">· manual</span>
													</Show>
												</TableCell>
												<TableCell class="py-2.5">
													<div class="flex items-center gap-2">
														<SourceBadge source={r.target.source} />
														{r.target.name}
													</div>
												</TableCell>
												<TableCell class="py-2.5">
													<RunStatusText run={r} />
													<Show when={r.stopReason === "page_cap"}>
														<span class="ml-1.5 text-xs text-status-interview">
															· cap
														</span>
													</Show>
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num value={r.pages} />
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num value={r.listed} />
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num
														value={r.fresh}
														class="font-semibold text-accent-text"
													/>
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num value={r.elsewhere} />
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num value={r.repeat} />
												</TableCell>
												<TableCell class="py-2.5 text-right">
													<Num
														value={r.errors.length}
														class="text-destructive-strong"
													/>
												</TableCell>
											</TableRow>
											<Show when={open() === r.id}>
												<tr class="border-b border-border">
													<td colSpan={9} class="p-0">
														<RunDetail run={r} />
													</td>
												</tr>
											</Show>
										</>
									)}
								</For>
							</TableBody>
						</Table>
					)}
				</Paged>
			</section>
		</>
	);
}

function VariantB(props: VariantProps) {
	const [open, setOpen] = createSignal<string>();
	const selected = () => TARGETS.find((t) => t.id === props.search().target);
	const groups = () =>
		(["search", "board"] as const)
			.filter((k) => !props.search().kind || props.search().kind === k)
			.map((k) => ({
				kind: k,
				targets: targetsSorted(
					(t) =>
						t.kind === k &&
						(!props.search().source || t.source === props.search().source),
				),
			}));
	const chartRuns = () => props.filtered().slice(0, 12).reverse();
	const maxListed = () => Math.max(1, ...chartRuns().map((r) => r.listed));

	return (
		<>
			<Filters {...props} hideTarget />
			<div class="grid gap-4 lg:grid-cols-[300px_1fr]">
				<nav class="self-start overflow-hidden rounded-xl border border-border bg-surface">
					<button
						type="button"
						class={cn(
							"flex w-full items-center justify-between px-4 py-2.5 text-left text-sm hover:bg-surface-muted",
							!selected() && "bg-accent-subtle text-accent-text",
						)}
						onClick={() => props.set({ target: undefined })}
					>
						All activity
						<span class="text-xs text-faint">{RUNS.length} runs</span>
					</button>
					<For each={groups()}>
						{(g) => (
							<>
								<p class="border-t border-border bg-surface-muted px-4 py-1.5 text-2xs font-semibold uppercase tracking-wider text-faint">
									{g.kind === "search" ? "Searches" : "Boards"}
								</p>
								<For each={g.targets}>
									{(t) => (
										<button
											type="button"
											class={cn(
												"flex w-full items-center gap-2.5 border-t border-border px-4 py-2.5 text-left hover:bg-surface-muted",
												selected()?.id === t.id && "bg-accent-subtle",
											)}
											onClick={() => props.set({ target: t.id })}
										>
											<HealthDot health={healthOf(t.id)} />
											<span class="min-w-0 flex-1">
												<span class="block truncate text-sm text-foreground">
													{t.name}
												</span>
												<span class="block text-xs text-faint">
													{t.source} · {healthReason(t.id)}
												</span>
											</span>
										</button>
									)}
								</For>
							</>
						)}
					</For>
				</nav>

				<section class="min-w-0 space-y-4">
					<Show
						when={selected()}
						fallback={
							<div class="grid grid-cols-3 gap-3">
								<For each={["red", "amber", "green"] as Health[]}>
									{(h) => (
										<div class="rounded-xl border border-border bg-surface p-4">
											<div class="flex items-center gap-2 text-xs text-faint">
												<HealthDot health={h} />
												{h === "red"
													? "Broken"
													: h === "amber"
														? "Needs a look"
														: "Healthy"}
											</div>
											<p class="mt-1 font-mono text-2xl font-semibold text-foreground">
												{TARGETS.filter((t) => healthOf(t.id) === h).length}
											</p>
										</div>
									)}
								</For>
							</div>
						}
					>
						{(t) => (
							<div class="rounded-xl border border-border bg-surface p-5">
								<div class="mb-4 flex items-center gap-2">
									<HealthDot health={healthOf(t().id)} />
									<SourceBadge source={t().source} />
									<h2 class="text-base font-semibold text-foreground">
										{t().name}
									</h2>
									<span class="ml-auto text-xs text-faint">
										{healthReason(t().id)}
									</span>
								</div>
								<p class="mb-2 text-xs text-faint">
									Last {chartRuns().length} runs: listings per run
								</p>
								<div class="flex h-32 items-end gap-1.5">
									<For each={chartRuns()}>
										{(r) => (
											<div
												class="flex flex-1 flex-col-reverse overflow-hidden rounded-sm"
												style={{
													height: `${Math.max(4, (r.listed / maxListed()) * 100)}%`,
												}}
												title={`${formatRelative(r.startedAt)}: ${r.fresh} new, ${r.elsewhere} elsewhere, ${r.repeat} repeat`}
											>
												<Show
													when={r.status === "failed"}
													fallback={
														<>
															<span
																class="bg-primary"
																style={{ flex: r.fresh }}
															/>
															<span
																class="bg-accent-border"
																style={{ flex: r.elsewhere }}
															/>
															<span
																class="bg-border-strong"
																style={{ flex: r.repeat }}
															/>
														</>
													}
												>
													<span class="h-full bg-destructive" />
												</Show>
											</div>
										)}
									</For>
								</div>
								<Legend />
							</div>
						)}
					</Show>

					<div class="overflow-hidden rounded-xl border border-border bg-surface">
						<Paged
							runs={props.filtered()}
							page={props.search().page ?? 1}
							set={props.set}
						>
							{(runs) => (
								<ul class="divide-y divide-border">
									<For each={runs}>
										{(r) => (
											<li>
												<button
													type="button"
													class="flex w-full items-center gap-4 px-4 py-2.5 text-left text-data hover:bg-surface-muted"
													onClick={() =>
														setOpen(open() === r.id ? undefined : r.id)
													}
												>
													<span class="w-24 shrink-0 text-xs text-muted">
														{formatRelative(r.startedAt)}
													</span>
													<Show when={!selected()}>
														<span class="w-56 shrink-0 truncate">
															{r.target.name}
														</span>
													</Show>
													<span class="w-24 shrink-0">
														<RunStatusText run={r} />
													</span>
													<span class="text-xs text-muted">
														<Num value={r.pages} /> pages ·{" "}
														<Num
															value={r.fresh}
															class="font-semibold text-accent-text"
														/>{" "}
														new · <Num value={r.elsewhere} /> elsewhere ·{" "}
														<Num value={r.repeat} /> repeat
														<Show when={r.target.kind === "board"}>
															{" "}
															· <Num value={r.removed} /> removed
														</Show>
													</span>
													<Show when={r.errors.length > 0}>
														<span class="ml-auto text-xs font-medium text-destructive-strong">
															{r.errors.length} error
															{r.errors.length > 1 ? "s" : ""}
														</span>
													</Show>
												</button>
												<Show when={open() === r.id}>
													<RunDetail run={r} />
												</Show>
											</li>
										)}
									</For>
								</ul>
							)}
						</Paged>
					</div>
				</section>
			</div>
		</>
	);
}

function Legend() {
	return (
		<div class="mt-3 flex gap-4 text-xs text-faint">
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-sm bg-primary" />
				New
			</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-sm bg-accent-border" />
				Found elsewhere
			</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-sm bg-border-strong" />
				Repeat
			</span>
			<span class="flex items-center gap-1.5">
				<span class="size-2 rounded-sm bg-destructive" />
				Failed
			</span>
		</div>
	);
}

function dayLabel(iso: string) {
	return new Date(iso).toLocaleDateString("en-GB", {
		weekday: "long",
		day: "numeric",
		month: "short",
	});
}

function VariantC(props: VariantProps) {
	const [open, setOpen] = createSignal<string>();
	const problems = () => targetsSorted((t) => healthOf(t.id) !== "green");
	const groupByDay = (runs: Run[]) => {
		const byDay = new Map<string, Run[]>();
		for (const r of runs)
			byDay.set(dayLabel(r.startedAt), [
				...(byDay.get(dayLabel(r.startedAt)) ?? []),
				r,
			]);
		return [...byDay.entries()];
	};

	return (
		<>
			<section class="mb-6">
				<div class="mb-2 flex items-baseline gap-3">
					<h2 class="text-sm font-semibold text-foreground">Needs attention</h2>
					<span class="text-xs text-faint">
						{problems().length} of {TARGETS.length} ·{" "}
						{TARGETS.length - problems().length} healthy
					</span>
				</div>
				<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
					<For each={problems()}>
						{(t) => (
							<button
								type="button"
								class={cn(
									"rounded-xl border bg-surface p-4 text-left transition-colors hover:border-border-strong",
									healthOf(t.id) === "red"
										? "border-destructive/40"
										: "border-border",
									props.search().target === t.id && "ring-2 ring-primary",
								)}
								onClick={() =>
									props.set({
										target: props.search().target === t.id ? undefined : t.id,
									})
								}
							>
								<div class="mb-2 flex items-center gap-2">
									<HealthDot health={healthOf(t.id)} />
									<SourceBadge source={t.source} />
									<span class="ml-auto">
										<LastRunsStrip targetId={t.id} />
									</span>
								</div>
								<p class="truncate text-sm font-medium text-foreground">
									{t.name}
								</p>
								<p
									class={cn(
										"text-xs",
										healthOf(t.id) === "red"
											? "text-destructive-strong"
											: "text-muted",
									)}
								>
									{healthReason(t.id)}
								</p>
								<p class="mt-2 truncate font-mono text-2xs text-faint">
									{RUNS.find((r) => r.target.id === t.id && r.errors.length)
										?.errors[0]?.message ?? ""}
								</p>
							</button>
						)}
					</For>
				</div>
			</section>

			<Filters {...props} />
			<Paged
				runs={props.filtered()}
				page={props.search().page ?? 1}
				pageSize={PAGE_SIZE * 2}
				set={props.set}
			>
				{(runs) => (
					<div class="mb-4 space-y-5">
						<For each={groupByDay(runs)}>
							{([day, dayRuns]) => (
								<section>
									<h3 class="mb-1.5 text-2xs font-semibold uppercase tracking-wider text-faint">
										{day}
									</h3>
									<ol class="overflow-hidden rounded-xl border border-border bg-surface">
										<For each={dayRuns}>
											{(r) => (
												<li class="border-b border-border last:border-0">
													<button
														type="button"
														class="flex w-full items-center gap-3 px-4 py-2 text-left hover:bg-surface-muted"
														onClick={() =>
															setOpen(open() === r.id ? undefined : r.id)
														}
													>
														<span class="w-12 shrink-0 font-mono text-xs text-faint">
															{new Date(r.startedAt).toLocaleTimeString(
																"en-GB",
																{ hour: "2-digit", minute: "2-digit" },
															)}
														</span>
														<span
															class={cn(
																"size-2 shrink-0 rounded-full",
																r.status === "failed"
																	? "bg-destructive"
																	: r.status === "running"
																		? "animate-pulse bg-primary"
																		: r.errors.length
																			? "bg-status-interview"
																			: "bg-status-offer",
															)}
														/>
														<span class="min-w-0 flex-1 truncate text-data text-foreground">
															<span class="text-muted">{r.target.source}</span>{" "}
															· {r.target.name}
														</span>
														<span class="hidden text-xs text-muted md:inline">
															<Show
																when={r.status !== "failed"}
																fallback={
																	<span class="text-destructive-strong">
																		failed: {r.errors.at(-1)?.message}
																	</span>
																}
															>
																<Num
																	value={r.fresh}
																	zero="no"
																	class="font-semibold text-accent-text"
																/>{" "}
																new of <Num value={r.listed} /> across{" "}
																<Num value={r.pages} /> page
																{r.pages > 1 ? "s" : ""}
																<Show when={r.errors.length}>
																	{" "}
																	· {r.errors.length} retried
																</Show>
															</Show>
														</span>
														<YieldBar run={r} class="hidden lg:flex" />
													</button>
													<Show when={open() === r.id}>
														<RunDetail run={r} />
													</Show>
												</li>
											)}
										</For>
									</ol>
								</section>
							)}
						</For>
					</div>
				)}
			</Paged>
		</>
	);
}
