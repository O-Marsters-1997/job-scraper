import { createFileRoute } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { ScoringConfig } from "../../../api/scoringConfig";
import type {
	ScoringOption,
	ScoringOptionsView,
} from "../../../api/scoringOptions";
import { useRecomputeScores, useScoringStatus } from "../../../hooks/useScores";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";
import { useScoringOptions } from "../../../hooks/useScoringOptions";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

type Stance = "nice" | "avoid" | "block";
type StanceMap = Record<string, Stance | undefined>;

const PAIR_COPY: Record<
	string,
	{ title: string; like: string; avoid: string; block?: string; help: string }
> = {
	role: {
		title: "Roles",
		like: "What kinds of role would you enjoy?",
		avoid: "What kinds of role would you avoid?",
		help: "A job matching any one role you'd enjoy gets the boost.",
	},
	tech: {
		title: "Technologies",
		like: "What technologies would you enjoy working with?",
		avoid: "What technologies would you avoid?",
		help: "Type to search all of them.",
	},
	domain: {
		title: "Industries",
		like: "Which industries would you enjoy working in?",
		avoid: "Which industries would you avoid?",
		block: "Which industries should be hidden completely?",
		help: "Type to search. Blocked industries are hidden from the jobs list, not just scored down.",
	},
};

const MULTI_COPY: Record<string, { title: string; help: string }> = {
	seniority: {
		title: "Which levels are you looking at?",
		help: "Most postings don't state a level. That never counts against a job.",
	},
	work: {
		title: "Which working arrangements suit you?",
		help: "Pick every arrangement you'd take.",
	},
	stage: {
		title: "Which company stages interest you?",
		help: "Pick every stage you'd consider.",
	},
};

const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	avoid: "border-destructive/40 bg-destructive-subtle text-destructive-strong",
	block: "border-destructive bg-destructive text-white",
};

function StancePicker(props: {
	dim: string;
	stance: Stance;
	label: string;
	options: ScoringOption[];
	stances: StanceMap;
	pick: (id: string, stance: Stance) => void;
	unpick: (id: string) => void;
}) {
	const [query, setQuery] = createSignal("");
	const inputId = () => `scoring-${props.dim}-${props.stance}`;
	const picked = createMemo(() =>
		props.options.filter((o) => props.stances[o.id] === props.stance),
	);
	const results = createMemo(() => {
		const q = query().trim().toLowerCase();
		if (!q) return [];
		return props.options
			.filter((o) => !props.stances[o.id] && o.label.toLowerCase().includes(q))
			.slice(0, 8);
	});

	return (
		<div>
			<label for={inputId()} class="block text-sm font-medium text-foreground">
				{props.label}
			</label>
			<div class="relative mt-1.5">
				<div class="field flex min-h-10 flex-wrap items-center gap-1.5 py-1.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/10">
					<For each={picked()}>
						{(o) => (
							<span
								class={`inline-flex items-center rounded-full border text-sm ${STANCE_TONE[props.stance]}`}
							>
								<span class="py-0.5 pl-2.5">{o.label}</span>
								<button
									type="button"
									aria-label={`Remove ${o.label}`}
									onClick={() => props.unpick(o.id)}
									class="px-2 py-0.5 opacity-70 hover:opacity-100"
								>
									×
								</button>
							</span>
						)}
					</For>
					<input
						id={inputId()}
						type="text"
						autocomplete="off"
						class="h-7 min-w-40 flex-1 bg-transparent text-sm outline-none placeholder:text-faint"
						placeholder={picked().length ? "Add another…" : "Search…"}
						value={query()}
						onInput={(e) => setQuery(e.currentTarget.value)}
						onKeyDown={(e) => {
							const first = results()[0];
							if (e.key === "Enter" && first) {
								e.preventDefault();
								props.pick(first.id, props.stance);
								setQuery("");
							}
							if (e.key === "Escape") setQuery("");
						}}
					/>
				</div>
				<Show when={query().trim()}>
					<ul class="absolute z-30 mt-1 w-full max-w-md divide-y divide-border rounded-lg border border-border bg-surface shadow-xl">
						<For
							each={results()}
							fallback={
								<li class="px-3 py-2 text-xs text-faint">No matches.</li>
							}
						>
							{(o) => (
								<li>
									<button
										type="button"
										onClick={() => {
											props.pick(o.id, props.stance);
											setQuery("");
										}}
										class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-surface-muted"
									>
										{o.label}
									</button>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</div>
		</div>
	);
}

function PairSection(props: {
	dim: string;
	options: ScoringOption[];
	stances: StanceMap;
	pick: (id: string, stance: Stance) => void;
	unpick: (id: string) => void;
}) {
	const copy = PAIR_COPY[props.dim];
	if (!copy) return null;
	return (
		<div class="px-5 py-4">
			<p class="text-xs text-faint">{copy.help}</p>
			<div class="mt-3 space-y-4">
				<StancePicker
					dim={props.dim}
					stance="nice"
					label={copy.like}
					options={props.options}
					stances={props.stances}
					pick={props.pick}
					unpick={props.unpick}
				/>
				<StancePicker
					dim={props.dim}
					stance="avoid"
					label={copy.avoid}
					options={props.options}
					stances={props.stances}
					pick={props.pick}
					unpick={props.unpick}
				/>
				<Show when={copy.block}>
					{(label) => (
						<StancePicker
							dim={props.dim}
							stance="block"
							label={label()}
							options={props.options}
							stances={props.stances}
							pick={props.pick}
							unpick={props.unpick}
						/>
					)}
				</Show>
			</div>
		</div>
	);
}

function MultiSection(props: {
	dim: string;
	options: ScoringOption[];
	stances: StanceMap;
	pick: (id: string, stance: Stance) => void;
	unpick: (id: string) => void;
}) {
	const copy = MULTI_COPY[props.dim];
	if (!copy) return null;
	return (
		<div class="px-5 py-4">
			<p class="text-sm font-medium text-foreground">{copy.title}</p>
			<p class="mt-0.5 text-xs text-faint">{copy.help}</p>
			<div class="mt-2.5 flex flex-wrap gap-2">
				<For each={props.options}>
					{(o) => {
						const active = () => props.stances[o.id] === "nice";
						return (
							<button
								type="button"
								aria-pressed={active()}
								onClick={() =>
									active() ? props.unpick(o.id) : props.pick(o.id, "nice")
								}
								class={`inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors ${
									active()
										? `${STANCE_TONE.nice} font-medium`
										: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground"
								}`}
							>
								<span
									aria-hidden="true"
									class={`grid size-3.5 place-items-center rounded-sm border text-2xs leading-none ${
										active()
											? "border-accent-text bg-accent-text text-primary-foreground"
											: "border-border-strong bg-surface"
									}`}
								>
									{active() ? "✓" : ""}
								</span>
								{o.label}
							</button>
						);
					}}
				</For>
			</div>
		</div>
	);
}

function ScoringPage() {
	const configQuery = useScoringConfig();
	const optionsQuery = useScoringOptions();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="Scoring settings"
				subtitle="Pick what you'd enjoy or avoid. Every posting is asked the same fixed questions, so changing a pick re-ranks every job for free."
			/>
			<QueryBoundary query={optionsQuery} fallbackRows={4}>
				{(options) => (
					<QueryBoundary query={configQuery} fallbackRows={4}>
						{(config) => <ScoringForm config={config} options={options} />}
					</QueryBoundary>
				)}
			</QueryBoundary>
		</div>
	);
}

function ScoringForm(props: {
	config: ScoringConfig;
	options: ScoringOptionsView;
}) {
	const mutation = useUpdateScoringConfig();
	const recompute = useRecomputeScores();
	const scoringStatus = useScoringStatus();

	const [stances, setStances] = createStore<StanceMap>(
		Object.fromEntries(
			props.config.preferences.picks.map((p) => [
				p.optionId,
				p.stance as Stance,
			]),
		),
	);
	const pick = (id: string, stance: Stance) => setStances(id, stance);
	const unpick = (id: string) => setStances(id, undefined);

	const [salaryFloor, setSalaryFloor] = createSignal(
		props.config.preferences.salaryFloor?.amount.toString() ?? "",
	);
	const [threshold, setThreshold] = createSignal(props.config.notifyThreshold);
	const [noGoTech, setNoGoTech] = createSignal(
		props.config.preferences.blockedTech.join(", "),
	);
	const [companies, setCompanies] = createSignal(
		props.config.excludedCompanies.join(", "),
	);
	const [locations, setLocations] = createSignal(
		props.config.excludedLocations.join(", "),
	);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const optionsFor = (dim: string) =>
		props.options.options.filter((o) => o.dimension === dim);

	const splitList = (value: string) =>
		value
			.split(",")
			.map((v) => v.trim())
			.filter((v) => v.length > 0);

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({
				notifyThreshold: threshold(),
				excludedCompanies: splitList(companies()),
				excludedLocations: splitList(locations()),
				preferences: {
					picks: Object.entries(stances)
						.filter((entry): entry is [string, Stance] => Boolean(entry[1]))
						.map(([optionId, stance]) => ({
							optionId,
							stance,
							source: "manual",
						})),
					salaryFloor: salaryFloor()
						? { amount: Number(salaryFloor()), currency: "GBP" }
						: null,
					blockedTech: splitList(noGoTech()),
				},
				updatedAt: props.config.updatedAt,
			});
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				<section aria-labelledby="scoring-role">
					<h2 id="scoring-role" class="text-base font-semibold text-foreground">
						Role &amp; working
					</h2>
					<div class="mt-3 divide-y divide-border rounded-xl border border-border bg-surface">
						<PairSection
							dim="role"
							options={optionsFor("role")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
						<MultiSection
							dim="seniority"
							options={optionsFor("seniority")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
						<MultiSection
							dim="work"
							options={optionsFor("work")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
					</div>
				</section>

				<section aria-labelledby="scoring-tech">
					<h2 id="scoring-tech" class="text-base font-semibold text-foreground">
						Technology
					</h2>
					<div class="mt-3 divide-y divide-border rounded-xl border border-border bg-surface">
						<PairSection
							dim="tech"
							options={optionsFor("tech")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
					</div>
				</section>

				<section aria-labelledby="scoring-company">
					<h2
						id="scoring-company"
						class="text-base font-semibold text-foreground"
					>
						Company
					</h2>
					<div class="mt-3 divide-y divide-border rounded-xl border border-border bg-surface">
						<MultiSection
							dim="stage"
							options={optionsFor("stage")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
						<PairSection
							dim="domain"
							options={optionsFor("domain")}
							stances={stances}
							pick={pick}
							unpick={unpick}
						/>
					</div>
				</section>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">
							Exclusion filters
						</p>
						<p class="mt-0.5 text-xs text-faint">
							Company and no-go tech are checked before any scoring runs, so a
							match never reaches Jev. Leave a field blank to exclude nothing on
							that axis.
						</p>
					</div>
					<div class="divide-y divide-border">
						<div class="px-5 py-4">
							<label
								for="no-go-tech"
								class="text-xs font-medium text-foreground"
							>
								No-go tech
							</label>
							<p class="mt-0.5 text-xs text-faint">
								Comma-separated. Matches whole words in the title or description
								(e.g. "java" won't reject "JavaScript").
							</p>
							<Input
								id="no-go-tech"
								value={noGoTech()}
								onInput={(e) => setNoGoTech(e.currentTarget.value)}
								placeholder="kubernetes, php"
								class="mt-2"
							/>
						</div>

						<div class="px-5 py-4">
							<label
								for="excluded-companies"
								class="text-xs font-medium text-foreground"
							>
								Excluded companies
							</label>
							<p class="mt-0.5 text-xs text-faint">Comma-separated.</p>
							<Input
								id="excluded-companies"
								value={companies()}
								onInput={(e) => setCompanies(e.currentTarget.value)}
								placeholder="Acme Corp"
								class="mt-2"
							/>
						</div>

						<div class="px-5 py-4">
							<label
								for="excluded-locations"
								class="text-xs font-medium text-foreground"
							>
								Excluded locations (discovery only)
							</label>
							<p class="mt-0.5 text-xs text-faint">
								Comma-separated. Stops these jobs being scanned for detail; it
								never affects a score you already have.
							</p>
							<Input
								id="excluded-locations"
								value={locations()}
								onInput={(e) => setLocations(e.currentTarget.value)}
								placeholder="United States"
								class="mt-2"
							/>
						</div>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Salary floor</p>
						<p class="mt-0.5 text-xs text-faint">
							A job below this costs points, but a job that doesn't state a
							salary is never penalised.
						</p>
					</div>
					<div class="px-5 py-4">
						<label
							for="salary-floor"
							class="text-xs font-medium text-foreground"
						>
							Minimum salary
						</label>
						<div class="mt-2 flex items-center gap-2">
							<span class="text-sm text-faint">£</span>
							<Input
								id="salary-floor"
								type="number"
								step="5000"
								min="0"
								value={salaryFloor()}
								onInput={(e) => setSalaryFloor(e.currentTarget.value)}
								placeholder="No floor"
								class="w-32 font-mono tabular-nums"
							/>
							<span class="text-xs text-faint">a year</span>
						</div>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Notifications</p>
						<p class="mt-0.5 text-xs text-faint">
							Tune when you're notified after a job is scored.
						</p>
					</div>
					<div class="px-5 py-4">
						<label for="threshold" class="text-xs font-medium text-foreground">
							Notify threshold
						</label>
						<p class="mt-0.5 text-xs text-faint">
							You only receive notifications for jobs scoring at or above this
							suitability score.
						</p>
						<div class="mt-2 flex items-center gap-2">
							<Input
								id="threshold"
								type="number"
								min="0"
								max="100"
								value={threshold()}
								onInput={(e) => setThreshold(Number(e.currentTarget.value))}
								class="w-20"
							/>
							<span class="text-xs text-faint">out of 100</span>
						</div>
					</div>
				</Card>

				<div class="flex items-center gap-3">
					<Button onClick={handleSave} disabled={mutation.isPending}>
						{mutation.isPending ? "Saving…" : "Save"}
					</Button>
				</div>

				<Card>
					<CardHeader>
						<CardTitle>Recompute scores</CardTitle>
					</CardHeader>
					<CardContent>
						<Show
							when={scoringStatus.data}
							fallback={
								<p>
									{scoringStatus.isError
										? "Could not load scoring status."
										: "Loading scoring status…"}
								</p>
							}
						>
							{(status) => <p>{status().pending} answer effects pending.</p>}
						</Show>
						<Button
							variant="outline"
							disabled={recompute.isPending}
							onClick={() => recompute.mutate()}
						>
							{recompute.isPending ? "Recomputing…" : "Recompute scores"}
						</Button>
						<Show when={recompute.data}>
							<p>{recompute.data?.recomputed} jobs re-ranked.</p>
						</Show>
						<Show when={recompute.isError}>
							<p>Could not recompute scores. Try again.</p>
						</Show>
					</CardContent>
				</Card>
			</div>
		</>
	);
}
