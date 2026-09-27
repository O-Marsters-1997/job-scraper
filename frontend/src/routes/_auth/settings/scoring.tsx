import { Tabs } from "@kobalte/core/tabs";
import { createFileRoute } from "@tanstack/solid-router";
import {
	createMemo,
	createSignal,
	For,
	type JSX,
	onCleanup,
	onMount,
	Show,
} from "solid-js";
import { createStore } from "solid-js/store";
import { FormFeedback } from "@/components/FormFeedback";
import { MultiCombobox } from "@/components/MultiCombobox";
import { PageLayout } from "@/components/PageLayout";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { uniqueCapitalised } from "@/lib/capitalise";
import { cn } from "@/lib/utils";
import type { ScoringConfig } from "../../../api/scoringConfig";
import type {
	ScoringOption,
	ScoringOptionsView,
} from "../../../api/scoringOptions";
import { useCompanies } from "../../../hooks/useCompanies";
import { useRecomputeScores, useScoringStatus } from "../../../hooks/useScores";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";
import { useScoringOptions } from "../../../hooks/useScoringOptions";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

const TITLE = "Scoring settings";

type Stance = "nice" | "avoid";
type StanceMap = Record<string, Stance | undefined>;
type Dim = ScoringOption["dimension"];

const SECTIONS = [
	{
		id: "role",
		title: "Role",
		summary: "Role type, salary, level, setup",
	},
	{
		id: "stack",
		title: "Tech & industry",
		summary: "Stack, industries, company stage",
	},
	{
		id: "filters",
		title: "Filters & alerts",
		summary: "Exclusions, notify threshold",
	},
	{
		id: "other",
		title: "Other details",
		summary: "Anything else, in your own words",
	},
] as const;

type SectionId = (typeof SECTIONS)[number]["id"];

const isTyping = (target: EventTarget | null) =>
	target instanceof HTMLElement &&
	(target.isContentEditable ||
		["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName));

const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	avoid: "border-destructive/40 bg-destructive-subtle text-destructive-strong",
};

function ScoringPage() {
	const configQuery = useScoringConfig();
	const optionsQuery = useScoringOptions();
	return (
		<QueryBoundary query={optionsQuery} fallbackRows={4}>
			{(options) => (
				<QueryBoundary query={configQuery} fallbackRows={4}>
					{(config) => <ScoringForm config={config} options={options} />}
				</QueryBoundary>
			)}
		</QueryBoundary>
	);
}

function Field(props: {
	label: string;
	for: string;
	hint?: string;
	children: JSX.Element;
}) {
	return (
		<div>
			<label for={props.for} class="block text-sm font-medium text-foreground">
				{props.label}
			</label>
			<Show when={props.hint}>
				<p class="mt-0.5 text-xs text-faint">{props.hint}</p>
			</Show>
			<div class="mt-2">{props.children}</div>
		</div>
	);
}

function ChoiceGroup(props: {
	legend: string;
	hint?: string;
	options: ScoringOption[];
	isOn: (id: string) => boolean;
	toggle: (id: string) => void;
}) {
	return (
		<fieldset>
			<legend class="text-sm font-medium text-foreground">
				{props.legend}
			</legend>
			<Show when={props.hint}>
				<p class="mt-0.5 text-xs text-faint">{props.hint}</p>
			</Show>
			<div class="mt-2 flex flex-wrap gap-2">
				<For each={props.options}>
					{(o) => (
						<button
							type="button"
							aria-pressed={props.isOn(o.id)}
							onClick={() => props.toggle(o.id)}
							class={cn(
								"inline-flex h-8 items-center gap-2 rounded-md border px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
								props.isOn(o.id)
									? cn(STANCE_TONE.nice, "font-medium")
									: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
							)}
						>
							<span
								aria-hidden="true"
								class={cn(
									"grid size-3.5 place-items-center rounded-sm border",
									props.isOn(o.id)
										? "border-accent-text bg-accent-text text-primary-foreground"
										: "border-border-strong bg-surface",
								)}
							>
								<Show when={props.isOn(o.id)}>
									<svg
										aria-hidden="true"
										width="10"
										height="10"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="3.5"
										stroke-linecap="round"
										stroke-linejoin="round"
									>
										<polyline points="20 6 9 17 4 12" />
									</svg>
								</Show>
							</span>
							{o.label}
						</button>
					)}
				</For>
			</div>
		</fieldset>
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
			props.config.preferences.picks
				.filter((p) => p.source === "manual")
				.map((p) => [p.optionId, p.stance as Stance]),
		),
	);

	const optionsFor = (dim: Dim) =>
		props.options.options.filter((o) => o.dimension === dim);
	const idsWith = (dim: Dim, stance: Stance) =>
		optionsFor(dim)
			.filter((o) => stances[o.id] === stance)
			.map((o) => o.id);
	const setStance = (dim: Dim, stance: Stance, ids: string[]) => {
		for (const o of optionsFor(dim)) {
			if (ids.includes(o.id)) setStances(o.id, stance);
			else if (stances[o.id] === stance) setStances(o.id, undefined);
		}
	};
	const isNice = (id: string) => stances[id] === "nice";
	const toggleNice = (id: string) =>
		setStances(id, isNice(id) ? undefined : "nice");

	const textPicks = createMemo(() =>
		props.config.preferences.picks.filter((p) => p.source === "text"),
	);
	const labelFor = (optionId: string) =>
		props.options.options.find((o) => o.id === optionId)?.label ?? optionId;

	const [preferenceText, setPreferenceText] = createSignal(
		props.config.preferences.preferenceText,
	);
	const [salaryFloor, setSalaryFloor] = createSignal(
		props.config.preferences.salaryFloor?.amount.toString() ?? "",
	);
	const [threshold, setThreshold] = createSignal(props.config.notifyThreshold);
	const [titleKeywords, setTitleKeywords] = createSignal(
		props.config.excludedTitleKeywords.join(", "),
	);
	const [companies, setCompanies] = createSignal(
		props.config.excludedCompanies,
	);
	const [locations, setLocations] = createSignal(
		props.config.excludedLocations,
	);
	const companiesQuery = useCompanies();
	const companyOptions = createMemo(() =>
		(companiesQuery.data ?? []).map((c) => ({ id: c.Name, label: c.Name })),
	);

	const [section, setSection] = createSignal<SectionId>("role");
	const triggers: Partial<Record<SectionId, HTMLButtonElement>> = {};
	onMount(() => {
		const cycle = (e: KeyboardEvent) => {
			if (e.metaKey || e.ctrlKey || e.altKey || isTyping(e.target)) return;
			const step = e.key === "]" ? 1 : e.key === "[" ? -1 : 0;
			if (!step) return;
			e.preventDefault();
			const i = SECTIONS.findIndex((s) => s.id === section());
			const next = SECTIONS[(i + step + SECTIONS.length) % SECTIONS.length];
			if (!next) return;
			setSection(next.id);
			triggers[next.id]?.focus();
		};
		document.addEventListener("keydown", cycle);
		onCleanup(() => document.removeEventListener("keydown", cycle));
	});
	const [notice, setNotice] = createSignal<string | null>(null);
	const [error, setError] = createSignal<string | null>(null);

	const flash = (message: string) => {
		setError(null);
		setNotice(message);
		setTimeout(() => setNotice(null), 3000);
	};

	const splitList = (value: string) =>
		value
			.split(",")
			.map((v) => v.trim())
			.filter((v) => v.length > 0);

	const handleSave = async () => {
		setNotice(null);
		setError(null);
		try {
			await mutation.mutateAsync({
				notifyThreshold: threshold(),
				excludedTitleKeywords: splitList(titleKeywords()),
				excludedCompanies: uniqueCapitalised(companies()),
				excludedLocations: uniqueCapitalised(locations()),
				preferences: {
					picks: Object.entries(stances)
						.filter((entry): entry is [string, Stance] => Boolean(entry[1]))
						.map(([optionId, stance]) => ({
							optionId,
							stance,
							source: "manual",
							overridden: false,
						})),
					salaryFloor: salaryFloor()
						? { amount: Number(salaryFloor()), currency: "GBP" }
						: null,
					preferenceText: preferenceText(),
				},
				updatedAt: props.config.updatedAt,
			});
			flash("Saved.");
		} catch {
			setError("Failed to save. Please try again.");
		}
	};

	const handleRecompute = async () => {
		setNotice(null);
		setError(null);
		try {
			const result = await recompute.mutateAsync();
			flash(`${result.recomputed} jobs re-ranked.`);
		} catch {
			setError("Could not recompute scores. Try again.");
		}
	};

	const pairPickers = (dim: Dim, like: string, avoid: string) => (
		<>
			<MultiCombobox
				label={like}
				options={optionsFor(dim).filter((o) => stances[o.id] !== "avoid")}
				value={idsWith(dim, "nice")}
				onChange={(ids) => setStance(dim, "nice", ids)}
				chipClass={STANCE_TONE.nice}
			/>
			<MultiCombobox
				label={avoid}
				options={optionsFor(dim).filter((o) => stances[o.id] !== "nice")}
				value={idsWith(dim, "avoid")}
				onChange={(ids) => setStance(dim, "avoid", ids)}
				chipClass={STANCE_TONE.avoid}
			/>
		</>
	);

	return (
		<PageLayout
			title={TITLE}
			actions={
				<>
					<Show when={scoringStatus.data?.pending}>
						{(pending) => (
							<span class="hidden text-xs text-faint tabular-nums sm:inline">
								{pending()} pending
							</span>
						)}
					</Show>
					<Button
						variant="outline"
						size="sm"
						disabled={recompute.isPending}
						onClick={handleRecompute}
					>
						{recompute.isPending ? "Recomputing…" : "Recompute scores"}
					</Button>
					<Button size="sm" disabled={mutation.isPending} onClick={handleSave}>
						{mutation.isPending ? "Saving…" : "Save"}
					</Button>
				</>
			}
		>
			<FormFeedback success={notice() ?? false} error={error()} />

			<Tabs
				orientation="vertical"
				value={section()}
				onChange={(v) => setSection(v as SectionId)}
				class="flex flex-col gap-6 md:flex-row md:items-start"
			>
				<Tabs.List class="scroll-slim flex gap-1 overflow-x-auto md:sticky md:top-20 md:w-60 md:shrink-0 md:flex-col md:overflow-visible">
					<For each={SECTIONS}>
						{(s) => (
							<Tabs.Trigger
								value={s.id}
								ref={(el: HTMLButtonElement) => {
									triggers[s.id] = el;
								}}
								class="min-w-40 shrink-0 rounded-lg border border-transparent px-3 py-2.5 text-left transition-colors hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary data-[selected]:border-border data-[selected]:bg-surface md:min-w-0"
							>
								<span class="block text-sm font-medium text-foreground">
									{s.title}
								</span>
								<span class="mt-0.5 block truncate text-xs text-faint">
									{s.summary}
								</span>
							</Tabs.Trigger>
						)}
					</For>
					<p class="mt-2 hidden px-3 text-xs text-faint md:block">
						Press <kbd class="font-mono">[</kbd> or{" "}
						<kbd class="font-mono">]</kbd> to switch sections
					</p>
				</Tabs.List>

				<div class="min-w-0 flex-1 self-stretch rounded-xl border border-border bg-surface md:min-h-[calc(100dvh-10rem-1px)]">
					<Tabs.Content value="role" class="flex flex-col gap-6 p-6">
						<ChoiceGroup
							legend="Which roles are you looking for?"
							options={optionsFor("role")}
							isOn={isNice}
							toggle={toggleNice}
						/>
						<Field
							label="Minimum salary"
							for="salary-floor"
							hint="Jobs that don't state a salary are never penalised."
						>
							<div class="relative w-44">
								<span class="pointer-events-none absolute inset-y-0 left-3 flex items-center text-sm text-faint">
									£
								</span>
								<Input
									id="salary-floor"
									type="number"
									step="5000"
									min="0"
									value={salaryFloor()}
									onInput={(e) => setSalaryFloor(e.currentTarget.value)}
									placeholder="No minimum"
									class="pl-7 font-mono tabular-nums"
								/>
							</div>
						</Field>
						<ChoiceGroup
							legend="Seniority"
							hint="Postings that don't state a level are never penalised."
							options={optionsFor("seniority")}
							isOn={isNice}
							toggle={toggleNice}
						/>
						<ChoiceGroup
							legend="Working arrangement"
							options={optionsFor("work")}
							isOn={isNice}
							toggle={toggleNice}
						/>
					</Tabs.Content>

					<Tabs.Content value="stack" class="flex flex-col gap-6 p-6">
						{pairPickers(
							"tech",
							"Technologies you'd enjoy",
							"Technologies to avoid",
						)}
						{pairPickers(
							"domain",
							"Industries you'd enjoy",
							"Industries to avoid",
						)}
						<ChoiceGroup
							legend="Company stage"
							options={optionsFor("stage")}
							isOn={isNice}
							toggle={toggleNice}
						/>
					</Tabs.Content>

					<Tabs.Content value="filters" class="flex flex-col gap-6 p-6">
						<p class="text-xs text-faint">
							Jobs matching any exclusion are dropped before scoring.
						</p>
						<Field
							label="Excluded title keywords"
							for="excluded-titles"
							hint={`Comma-separated, whole words only: "java" won't exclude "JavaScript".`}
						>
							<Input
								id="excluded-titles"
								value={titleKeywords()}
								onInput={(e) => setTitleKeywords(e.currentTarget.value)}
								placeholder="recruiter, sales, .net"
							/>
						</Field>
						<MultiCombobox
							label="Excluded companies"
							options={companyOptions()}
							value={companies()}
							onChange={(names) => setCompanies(uniqueCapitalised(names))}
							placeholder="Search or type a company…"
							chipClass={STANCE_TONE.avoid}
							creatable
						/>
						<MultiCombobox
							label="Excluded locations"
							options={[]}
							value={locations()}
							onChange={(names) => setLocations(uniqueCapitalised(names))}
							placeholder="Type a location, then Enter"
							chipClass={STANCE_TONE.avoid}
							creatable
						/>
						<Field
							label="Notify me at a score of"
							for="threshold"
							hint="Out of 100."
						>
							<Input
								id="threshold"
								type="number"
								min="0"
								max="100"
								value={threshold()}
								onInput={(e) => setThreshold(Number(e.currentTarget.value))}
								class="w-24 font-mono tabular-nums"
							/>
						</Field>
					</Tabs.Content>

					<Tabs.Content value="other" class="flex flex-col gap-6 p-6">
						<Field
							label="Anything else worth mentioning?"
							for="preference-text"
							hint="Whatever matches an option elsewhere becomes a pick. Your own picks always win."
						>
							<Textarea
								id="preference-text"
								value={preferenceText()}
								onInput={(e) => setPreferenceText(e.currentTarget.value)}
								placeholder="e.g. I'd like to work with people more senior than me, and avoid on-call rotations."
								class="min-h-32"
							/>
						</Field>
						<Show when={textPicks().length > 0}>
							<div class="flex flex-wrap gap-1.5">
								<For each={textPicks()}>
									{(p) => (
										<span
											class={cn(
												"inline-flex h-6 items-center rounded-full border px-2.5 text-xs font-medium",
												p.overridden
													? "border-border text-faint line-through"
													: (STANCE_TONE[p.stance as Stance] ??
															"border-border-strong bg-surface-muted text-muted"),
											)}
										>
											{labelFor(p.optionId)}
											{p.overridden && " (overridden)"}
										</span>
									)}
								</For>
							</div>
						</Show>
					</Tabs.Content>
				</div>
			</Tabs>
		</PageLayout>
	);
}
