import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import type {
	ScoringConfig,
	ScoringCriterion,
} from "../../../api/scoringConfig";
import { useQueueRescore, useScoringStatus } from "../../../hooks/useScores";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

const STARTER_PROFILE =
	"I am a software engineer with 3+ years of experience in backend development. I am looking for roles that involve: Go or Python, distributed systems or APIs, remote or hybrid work. I prefer companies with fewer than 500 employees. I am not interested in roles focused primarily on JavaScript frontend or mobile development.";

const DEFAULT_SCALE = [
	"Not relevant",
	"Weak",
	"Possible",
	"Strong",
	"Apply today",
];

const EMPTY_CRITERION: ScoringCriterion = {
	key: "",
	instructions: "",
	true: "",
	false: "",
	required: false,
};

// Mirrors score.SeniorityLevels in internal/score/filter.go — kept in sync manually.
const SENIORITY_LEVELS = [
	"intern",
	"junior",
	"mid",
	"senior",
	"staff",
	"principal",
	"lead",
	"manager",
	"director",
];

const PlusIcon = () => (
	<svg
		aria-hidden="true"
		width="12"
		height="12"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2.5"
		stroke-linecap="round"
	>
		<line x1="12" y1="5" x2="12" y2="19" />
		<line x1="5" y1="12" x2="19" y2="12" />
	</svg>
);

function ScoringPage() {
	const query = useScoringConfig();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="Scoring settings"
				subtitle="Filter out obvious non-fits, then describe what a good match looks like."
			/>
			<QueryBoundary query={query} fallbackRows={4}>
				{(data) => <ScoringForm data={data} />}
			</QueryBoundary>
		</div>
	);
}

// Extracted so signals initialize from resolved data once — background
// refetches don't clobber values the user is actively editing.
function ScoringForm(props: { data: ScoringConfig }) {
	const mutation = useUpdateScoringConfig();
	const scoringStatus = useScoringStatus();
	const rescore = useQueueRescore();
	const [profile, setProfile] = createSignal(
		props.data.scoringQuestions.profile ?? "",
	);
	const [criteria, setCriteria] = createStore<ScoringCriterion[]>(
		props.data.scoringQuestions.criteria,
	);
	const [scale, setScale] = createStore<string[]>(
		props.data.scoringQuestions.scale.length > 0
			? props.data.scoringQuestions.scale
			: DEFAULT_SCALE,
	);
	const [threshold, setThreshold] = createSignal(
		props.data.notifyThreshold ?? 70,
	);
	const [titleKeywords, setTitleKeywords] = createSignal(
		props.data.excludedTitleKeywords.join(", "),
	);
	const [companies, setCompanies] = createSignal(
		props.data.excludedCompanies.join(", "),
	);
	const [locations, setLocations] = createSignal(
		props.data.excludedLocations.join(", "),
	);
	const [seniority, setSeniority] = createSignal<string[]>(
		props.data.excludedSeniority,
	);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const toggleSeniority = (level: string) => {
		setSeniority((current) =>
			current.includes(level)
				? current.filter((l) => l !== level)
				: [...current, level],
		);
	};

	const splitList = (value: string) =>
		value
			.split(",")
			.map((v) => v.trim())
			.filter((v) => v.length > 0);

	const addCriterion = () =>
		setCriteria(criteria.length, { ...EMPTY_CRITERION });
	const removeCriterion = (index: number) =>
		setCriteria((current) => current.filter((_, i) => i !== index));

	const addScaleLevel = () => setScale(scale.length, "");
	const removeScaleLevel = (index: number) =>
		setScale((current) => current.filter((_, i) => i !== index));
	const moveScaleLevel = (index: number, delta: number) => {
		const target = index + delta;
		if (target < 0 || target >= scale.length) return;
		setScale((current) => {
			const next = [...current];
			const moved = next[index];
			const displaced = next[target];
			if (moved === undefined || displaced === undefined) return current;
			next[index] = displaced;
			next[target] = moved;
			return next;
		});
	};

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({
				notifyThreshold: threshold(),
				excludedTitleKeywords: splitList(titleKeywords()),
				excludedCompanies: splitList(companies()),
				excludedSeniority: seniority(),
				excludedLocations: splitList(locations()),
				scoringQuestions: {
					profile: profile(),
					criteria: [...criteria],
					scale: [...scale],
				},
			});
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	const handleUseTemplate = () => {
		setProfile(STARTER_PROFILE);
	};

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<label
							for="profile"
							class="text-base font-semibold text-foreground"
						>
							Candidate profile
						</label>
						<p class="mt-0.5 text-xs text-faint">
							Describe your ideal role. Used alongside your criteria and scale
							when scoring each job.
						</p>
					</div>
					<div class="px-5 py-4">
						<Show when={!profile()}>
							<div class="mb-4 rounded-lg border border-border bg-surface-muted px-4 py-3 text-sm text-muted">
								<p class="font-medium text-foreground">No profile set yet</p>
								<p class="mt-1 text-xs text-faint">
									Without a profile, jobs have no baseline to score against. Add
									one so jobs are ranked against your actual goals and
									experience.
								</p>
							</div>
						</Show>
						<textarea
							id="profile"
							rows={7}
							value={profile()}
							onInput={(e) => setProfile(e.currentTarget.value)}
							placeholder="Describe your background, preferred stack, work style, company size, location preferences…"
							class="w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
						/>
						<Show when={!profile()}>
							<button
								type="button"
								onClick={handleUseTemplate}
								class="mt-2 inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-xs font-medium text-muted transition hover:border-border-strong hover:text-foreground"
							>
								Use starter template
							</button>
						</Show>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="flex items-center justify-between border-b border-border px-5 py-4">
						<div>
							<p class="text-base font-semibold text-foreground">Criteria</p>
							<p class="mt-0.5 text-xs text-faint">
								Specific yes/no questions the scorer checks per job. Mark one
								required to sink the score on a clear miss.
							</p>
						</div>
					</div>
					<div class="divide-y divide-border">
						<For each={criteria}>
							{(criterion, index) => (
								<div class="flex flex-col gap-2 px-5 py-4">
									<div class="flex items-center justify-between gap-2">
										<Input
											value={criterion.key}
											onInput={(e) =>
												setCriteria(index(), "key", e.currentTarget.value)
											}
											placeholder="key (e.g. go_backend)"
											aria-label="Criterion key"
											class="max-w-[240px] font-mono text-xs"
										/>
										<button
											type="button"
											onClick={() => removeCriterion(index())}
											class="rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle"
										>
											Delete
										</button>
									</div>
									<Input
										value={criterion.instructions}
										onInput={(e) =>
											setCriteria(
												index(),
												"instructions",
												e.currentTarget.value,
											)
										}
										placeholder="What should the scorer check? e.g. Does the role use Go?"
										aria-label="Criterion instructions"
									/>
									<div class="flex gap-2">
										<Input
											value={criterion.true}
											onInput={(e) =>
												setCriteria(index(), "true", e.currentTarget.value)
											}
											placeholder="What a 'true' answer looks like"
											aria-label="Criterion true wording"
											class="flex-1"
										/>
										<Input
											value={criterion.false}
											onInput={(e) =>
												setCriteria(index(), "false", e.currentTarget.value)
											}
											placeholder="What a 'false' answer looks like"
											aria-label="Criterion false wording"
											class="flex-1"
										/>
									</div>
									<Switch
										checked={criterion.required}
										onChange={(checked: boolean) =>
											setCriteria(index(), "required", checked)
										}
									>
										<SwitchLabel class="inline-flex items-center gap-2">
											<SwitchControl>
												<SwitchThumb />
											</SwitchControl>
											Required
										</SwitchLabel>
									</Switch>
								</div>
							)}
						</For>
					</div>
					<div class="px-5 py-4">
						<Button variant="outline" size="sm" onClick={addCriterion}>
							<PlusIcon />
							Add criterion
						</Button>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Scale</p>
						<p class="mt-0.5 text-xs text-faint">
							Ordered from worst to best fit. At least 2 levels are required.
						</p>
					</div>
					<div class="divide-y divide-border">
						<For each={scale}>
							{(level, index) => (
								<div class="flex items-center gap-2 px-5 py-3">
									<span class="w-5 font-mono text-2xs text-faint">
										{index() + 1}
									</span>
									<Input
										value={level}
										onInput={(e) => setScale(index(), e.currentTarget.value)}
										aria-label={`Scale level ${index() + 1}`}
										class="flex-1"
									/>
									<button
										type="button"
										onClick={() => moveScaleLevel(index(), -1)}
										disabled={index() === 0}
										class="rounded px-1.5 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground disabled:opacity-30"
										aria-label="Move level up"
									>
										↑
									</button>
									<button
										type="button"
										onClick={() => moveScaleLevel(index(), 1)}
										disabled={index() === scale.length - 1}
										class="rounded px-1.5 py-1 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground disabled:opacity-30"
										aria-label="Move level down"
									>
										↓
									</button>
									<button
										type="button"
										onClick={() => removeScaleLevel(index())}
										disabled={scale.length <= 2}
										class="rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle disabled:opacity-30"
									>
										Delete
									</button>
								</div>
							)}
						</For>
					</div>
					<div class="px-5 py-4">
						<Button variant="outline" size="sm" onClick={addScaleLevel}>
							<PlusIcon />
							Add level
						</Button>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">
							Exclusion filters
						</p>
						<p class="mt-0.5 text-xs text-faint">
							Jobs matching any of these are dropped before AI scoring runs.
							Leave a field blank to exclude nothing on that axis.
						</p>
					</div>
					<div class="divide-y divide-border">
						<div class="px-5 py-4">
							<label
								for="excluded-titles"
								class="text-xs font-medium text-foreground"
							>
								Excluded title keywords
							</label>
							<p class="mt-0.5 text-xs text-faint">
								Comma-separated. Matches whole words in the title only (e.g.
								"java" won't reject "JavaScript").
							</p>
							<Input
								id="excluded-titles"
								value={titleKeywords()}
								onInput={(e) => setTitleKeywords(e.currentTarget.value)}
								placeholder="recruiter, sales, .net"
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
								Excluded locations
							</label>
							<p class="mt-0.5 text-xs text-faint">Comma-separated.</p>
							<Input
								id="excluded-locations"
								value={locations()}
								onInput={(e) => setLocations(e.currentTarget.value)}
								placeholder="United States"
								class="mt-2"
							/>
						</div>

						<div class="px-5 py-4">
							<p class="text-xs font-medium text-foreground">
								Excluded seniority levels
							</p>
							<p class="mt-0.5 text-xs text-faint">
								Only rejects titles that clearly signal one of these levels —
								ambiguous titles pass.
							</p>
							<div class="mt-2 flex flex-wrap gap-2">
								<For each={SENIORITY_LEVELS}>
									{(level) => (
										<Button
											type="button"
											variant={
												seniority().includes(level) ? "secondary" : "outline"
											}
											size="sm"
											onClick={() => toggleSeniority(level)}
										>
											{level}
										</Button>
									)}
								</For>
							</div>
						</div>
					</div>
				</Card>

				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Notifications</p>
						<p class="mt-0.5 text-xs text-faint">
							Tune when you're notified after AI scoring.
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
						<CardTitle>Existing assessments</CardTitle>
					</CardHeader>
					<CardContent>
						<Show
							when={scoringStatus.data}
							fallback={
								<p>
									{scoringStatus.isError
										? "Could not load assessment status."
										: "Loading assessment status…"}
								</p>
							}
						>
							{(status) => (
								<p>
									{status().stale} stale · {status().pending} pending ·{" "}
									{status().failed} failed
								</p>
							)}
						</Show>
						<Button
							variant="outline"
							disabled={rescore.isPending}
							onClick={() => rescore.mutate()}
						>
							{rescore.isPending ? "Queueing…" : "Rescore existing jobs"}
						</Button>
						<Show when={rescore.data}>
							<p>
								{rescore.data?.queued} jobs queued. Run again to queue the next
								batch.
							</p>
						</Show>
						<Show when={rescore.isError}>
							<p>Could not queue a rescore. Try again.</p>
						</Show>
					</CardContent>
				</Card>
			</div>
		</>
	);
}
