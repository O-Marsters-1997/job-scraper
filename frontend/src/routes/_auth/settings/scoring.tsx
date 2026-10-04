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
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";
import { useScoringOptions } from "../../../hooks/useScoringOptions";
import type { ScoringConfig, Stance } from "../../../types/scoringConfig";
import type {
	ScoringOption,
	ScoringOptionsView,
} from "../../../types/scoringOptions";
import { FiltersSection } from "./-scoring/FiltersSection";
import { RecomputeControls } from "./-scoring/RecomputeControls";
import { STANCE_LABEL, STANCE_TONE } from "./-scoring/stance";
import { useExclusionFilters } from "./-scoring/useExclusionFilters";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

type StanceMap = Record<string, Stance | undefined>;

const PICKER_STANCES: (Stance | undefined)[] = ["nice", "ok", "avoid"];
type Dim = ScoringOption["dimension"];

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

	const initialConfig = untrack(() => props.config());

	const [stances, setStances] = createStore<StanceMap>(
		Object.fromEntries(
			initialConfig.preferences.picks
				.filter((p) => p.source === "manual")
				.map((p) => [p.optionId, p.stance]),
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
	const choiceOf = (id: string) => {
		const stance = stances[id];
		return stance && { label: STANCE_LABEL[stance], tone: STANCE_TONE[stance] };
	};
	const cycleStance = (dim: Dim) => (id: string) => {
		const order = [
			undefined,
			...(props.options().dimensions.find((d) => d.key === dim)?.stances ?? []),
		];
		setStances(id, order[(order.indexOf(stances[id]) + 1) % order.length]);
	};

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
	const filters = useExclusionFilters(initialConfig);

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
				notifyThreshold: filters.threshold(),
				excludedTitleKeywords: filters.titleKeywords(),
				excludedCompanies: uniqueCapitalised(filters.companies()),
				excludedLocations: uniqueCapitalised(filters.locations()),
				requiredLocations: uniqueCapitalised(filters.requiredLocations()),
				requiredTitleKeywords: filters.requiredTitleKeywords(),
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

	const stancePicker = (dim: Dim, stance: Stance, label: string) => (
		<MultiCombobox
			label={label}
			options={optionsFor(dim).filter(
				(o) =>
					stances[o.id] === stance || !PICKER_STANCES.includes(stances[o.id]),
			)}
			value={idsWith(dim, stance)}
			onChange={(ids) => setStance(dim, stance, ids)}
			chipClass={STANCE_TONE[stance]}
		/>
	);
	const pairPickers = (dim: Dim, noun: string, want: string) => (
		<>
			{stancePicker(dim, "nice", `${noun} you most want to ${want}`)}
			{stancePicker(dim, "ok", `${noun} you'd be happy with`)}
			{stancePicker(dim, "avoid", `${noun} to avoid`)}
		</>
	);

	return (
		<>
			<SettingsActions>
				<RecomputeControls
					onStart={() => {
						setNotice(null);
						setError(null);
					}}
					onDone={flash}
					onError={setError}
				/>
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
						choiceOf={choiceOf}
						cycle={cycleStance("role")}
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
						choiceOf={choiceOf}
						cycle={cycleStance("seniority")}
					/>
					<ChoiceGroup
						legend="Employment type"
						options={optionsFor("employment")}
						choiceOf={choiceOf}
						cycle={cycleStance("employment")}
					/>
					<ChoiceGroup
						legend="Working arrangement"
						options={optionsFor("work")}
						choiceOf={choiceOf}
						cycle={cycleStance("work")}
					/>
				</Match>
				<Match when={params().section === "stack"}>
					{pairPickers("tech", "Technologies", "work with")}
					{pairPickers("domain", "Industries", "work in")}
					<ChoiceGroup
						legend="Company stage"
						options={optionsFor("stage")}
						choiceOf={choiceOf}
						cycle={cycleStance("stage")}
					/>
					<ChoiceGroup
						legend="Company size"
						options={optionsFor("size")}
						choiceOf={choiceOf}
						cycle={cycleStance("size")}
					/>
				</Match>
				<Match when={params().section === "filters"}>
					<FiltersSection filters={filters} />
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
												: STANCE_TONE[p.stance],
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
