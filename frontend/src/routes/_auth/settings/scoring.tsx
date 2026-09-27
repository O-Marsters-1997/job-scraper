import { createFileRoute, useParams } from "@tanstack/solid-router";
import type { Accessor } from "solid-js";
import {
	createMemo,
	createSignal,
	For,
	Match,
	Show,
	Switch,
	untrack,
} from "solid-js";
import { createStore } from "solid-js/store";
import { ChoiceGroup, Field } from "@/components/Field";
import { FormFeedback } from "@/components/FormFeedback";
import { MultiCombobox } from "@/components/MultiCombobox";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { uniqueCapitalised } from "@/lib/capitalise";
import { cn } from "@/lib/utils";
import { useCompanies } from "../../../hooks/useCompanies";
import { useRecomputeScores, useScoringStatus } from "../../../hooks/useScores";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";
import { useScoringOptions } from "../../../hooks/useScoringOptions";
import type { ScoringConfig } from "../../../types/scoringConfig";
import type {
	ScoringOption,
	ScoringOptionsView,
} from "../../../types/scoringOptions";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

type Stance = "nice" | "avoid";
type StanceMap = Record<string, Stance | undefined>;
type Dim = ScoringOption["dimension"];

const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	avoid: "border-destructive/40 bg-surface text-destructive-strong",
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

function ScoringForm(props: {
	config: Accessor<ScoringConfig>;
	options: Accessor<ScoringOptionsView>;
}) {
	const mutation = useUpdateScoringConfig();
	const recompute = useRecomputeScores();
	const scoringStatus = useScoringStatus();

	const initialConfig = untrack(() => props.config());

	const [stances, setStances] = createStore<StanceMap>(
		Object.fromEntries(
			initialConfig.preferences.picks
				.filter((p) => p.source === "manual")
				.map((p) => [p.optionId, p.stance as Stance]),
		),
	);

	const optionsFor = (dim: Dim) =>
		props.options().options.filter((o) => o.dimension === dim);
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
		props.config().preferences.picks.filter((p) => p.source === "text"),
	);
	const labelFor = (optionId: string) =>
		props.options().options.find((o) => o.id === optionId)?.label ?? optionId;

	const [preferenceText, setPreferenceText] = createSignal(
		initialConfig.preferences.preferenceText,
	);
	const [salaryFloor, setSalaryFloor] = createSignal(
		initialConfig.preferences.salaryFloor?.amount.toString() ?? "",
	);
	const [threshold, setThreshold] = createSignal(initialConfig.notifyThreshold);
	const [titleKeywords, setTitleKeywords] = createSignal(
		initialConfig.excludedTitleKeywords,
	);
	const [companies, setCompanies] = createSignal(
		initialConfig.excludedCompanies,
	);
	const [locations, setLocations] = createSignal(
		initialConfig.excludedLocations,
	);
	const companiesQuery = useCompanies();
	const companyOptions = createMemo(() =>
		(companiesQuery.data ?? []).map((c) => ({ id: c.Name, label: c.Name })),
	);

	const params = useParams({ strict: false });
	const [notice, setNotice] = createSignal<string | null>(null);
	const [error, setError] = createSignal<string | null>(null);

	const flash = (message: string) => {
		setError(null);
		setNotice(message);
		setTimeout(() => setNotice(null), 3000);
	};

	const handleSave = async () => {
		setNotice(null);
		setError(null);
		try {
			const result = await mutation.mutateAsync({
				notifyThreshold: threshold(),
				excludedTitleKeywords: titleKeywords(),
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
				updatedAt: props.config().updatedAt,
			});
			flash(
				result.backfillQueued > 0
					? `Saved – answering new picks for ${result.backfillQueued} jobs.`
					: "Saved.",
			);
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
		<>
			<SettingsActions>
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
			</SettingsActions>
			<FormFeedback success={notice() ?? false} error={error()} />
			<Switch>
				<Match when={params().section === "role"}>
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
				</Match>
				<Match when={params().section === "stack"}>
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
				</Match>
				<Match when={params().section === "filters"}>
					<p class="text-xs text-faint">
						Jobs matching any exclusion are dropped before scoring.
					</p>
					<MultiCombobox
						label="Excluded title keywords"
						hint={`Whole words only: "java" won't exclude "JavaScript".`}
						options={[]}
						value={titleKeywords()}
						onChange={setTitleKeywords}
						placeholder="Type a keyword, then Enter"
						chipClass={STANCE_TONE.avoid}
						creatable
					/>
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
				</Match>
				<Match when={params().section === "other"}>
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
				</Match>
			</Switch>
		</>
	);
}
