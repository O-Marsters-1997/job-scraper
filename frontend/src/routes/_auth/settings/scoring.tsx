import { createFileRoute, useParams } from "@tanstack/solid-router";
import type { Accessor, JSX } from "solid-js";
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
import { Field } from "@/components/Field";
import { FormFeedback } from "@/components/FormFeedback";
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
import { useScoringPreview } from "../../../hooks/useScoringPreview";
import type {
	Preferences,
	ScoringConfig,
	Stance,
} from "../../../types/scoringConfig";
import type {
	ScoringOption,
	ScoringOptionsView,
} from "../../../types/scoringOptions";
import { CustomOptions } from "./-scoring/CustomOptions";
import { FiltersSection } from "./-scoring/FiltersSection";
import { RatingKey } from "./-scoring/RatingKey";
import { RecomputeControls } from "./-scoring/RecomputeControls";
import { ScorePreview } from "./-scoring/ScorePreview";
import { SeniorityLadder } from "./-scoring/SeniorityLadder";
import { StanceRow } from "./-scoring/StanceRow";
import { STANCE_TONE } from "./-scoring/stance";
import { TechNeeded } from "./-scoring/TechNeeded";
import { useExclusionFilters } from "./-scoring/useExclusionFilters";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

type StanceMap = Record<string, Stance | undefined>;

const RATED_SECTIONS = ["role", "stack", "work", "custom"];
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
				.filter((p) => p.source === "manual" && p.weight === undefined)
				.map((p) => [p.optionId, p.stance]),
		),
	);
	const [weights, setWeights] = createStore<Record<string, number>>(
		Object.fromEntries(
			initialConfig.preferences.picks
				.filter((p) => p.source === "manual" && p.weight !== undefined)
				.map((p) => [p.optionId, p.weight ?? 0]),
		),
	);

	const optionsFor = (dim: Dim) =>
		props.options().options.filter((o) => o.dimension === dim);
	const rows = (dim: Dim) => {
		const allowed = () =>
			props.options().dimensions.find((d) => d.key === dim)?.stances ?? [];
		return (
			<div class="flex flex-col">
				<For each={optionsFor(dim)}>
					{(o) => (
						<StanceRow
							option={o}
							stance={stances[o.id]}
							allowed={allowed()}
							onChange={(s) => setStances(o.id, s)}
						/>
					)}
				</For>
			</div>
		);
	};
	const section = (title: string, children: JSX.Element) => (
		<section class="flex flex-col gap-3">
			<h2 class="text-base font-semibold text-foreground">{title}</h2>
			{children}
		</section>
	);

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
	const [techNeeded, setTechNeeded] = createSignal(
		initialConfig.preferences.techNeeded,
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

	const preferences = (): Preferences => ({
		picks: [
			...Object.entries(stances)
				.filter((entry): entry is [string, Stance] => Boolean(entry[1]))
				.map(([optionId, stance]) => ({
					optionId,
					stance,
					source: "manual",
					overridden: false,
				})),
			...Object.entries(weights)
				.filter(([, weight]) => weight > 0)
				.map(([optionId, weight]) => ({
					optionId,
					stance: "nice" as const,
					weight,
					source: "manual",
					overridden: false,
				})),
		],
		salaryFloor: salaryFloor()
			? { amount: Number(salaryFloor()), currency: "GBP" }
			: null,
		preferenceText: preferenceText(),
		techNeeded: techNeeded(),
	});

	const [previewOpen, setPreviewOpen] = createSignal(false);
	const preview = useScoringPreview(preferences, previewOpen);

	const handleSave = async () => {
		setNotice(null);
		setError(null);
		try {
			const result = await mutation.mutateAsync({
				notifyThreshold: filters.threshold(),
				maxJobAgeDays: filters.maxJobAgeDays(),
				excludedTitleKeywords: filters.titleKeywords(),
				excludedCompanies: uniqueCapitalised(filters.companies()),
				excludedLocations: uniqueCapitalised(filters.locations()),
				requiredLocations: uniqueCapitalised(filters.requiredLocations()),
				requiredTitleKeywords: filters.requiredTitleKeywords(),
				preferences: preferences(),
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
				<Button
					size="sm"
					variant="outline"
					aria-expanded={previewOpen()}
					onClick={() => setPreviewOpen((open) => !open)}
				>
					Preview
				</Button>
				<Button size="sm" loading={mutation.isPending} onClick={handleSave}>
					Save
				</Button>
			</SettingsActions>
			<FormFeedback success={notice() ?? false} error={error()} />
			<div
				class={cn(
					"flex flex-col gap-6",
					previewOpen() && "xl:grid xl:grid-cols-[3fr_2fr] xl:items-start",
				)}
			>
				<Show when={previewOpen()}>
					<div class="xl:sticky xl:top-20 xl:col-start-2 xl:row-start-1 xl:max-h-[calc(100dvh-6rem)] xl:overflow-y-auto">
						<ScorePreview query={preview} />
					</div>
				</Show>
				<div class="flex min-w-0 flex-col gap-6 xl:col-start-1 xl:row-start-1">
					<Show when={RATED_SECTIONS.includes(params().section ?? "")}>
						<RatingKey />
					</Show>
					<Switch>
						<Match when={params().section === "role"}>
							{section("Role", rows("role"))}
							<SeniorityLadder
								tiers={optionsFor("seniority")}
								weightOf={(id) => weights[id] ?? 0}
								setWeight={(id, weight) => setWeights(id, weight)}
							/>
							{section(
								"Pay and contract",
								<>
									<Field
										label="Minimum salary"
										for="salary-floor"
										hint="Jobs that don't state a salary, or state one in another currency, aren't affected. A stated salary below this caps the score."
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
									{rows("employment")}
								</>,
							)}
						</Match>
						<Match when={params().section === "stack"}>
							{section(
								"Tech stack",
								<>
									<TechNeeded value={techNeeded()} onChange={setTechNeeded} />
									{rows("tech")}
								</>,
							)}
							{section("Industries", rows("domain"))}
						</Match>
						<Match when={params().section === "work"}>
							{section(
								"Working arrangement",
								<>
									<p class="text-xs text-faint">
										To prefer some office days over others, rate the day counts
										instead of Hybrid.
									</p>
									{rows("work")}
								</>,
							)}
							{section("Company size", rows("size"))}
							{section("Company stage", rows("stage"))}
						</Match>
						<Match when={params().section === "custom"}>
							<CustomOptions
								options={props.options()}
								stanceOf={(id) => stances[id]}
								onRate={(id, s) => setStances(id, s)}
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
				</div>
			</div>
		</>
	);
}
