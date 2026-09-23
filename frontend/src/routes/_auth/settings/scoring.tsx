import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { ScoringConfig } from "../../../api/scoringConfig";
import { useQueueRescore, useScoringStatus } from "../../../hooks/useScores";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

const STARTER_RUBRIC =
	"I am a software engineer with 3+ years of experience in backend development. I am looking for roles that involve: Go or Python, distributed systems or APIs, remote or hybrid work. I prefer companies with fewer than 500 employees. I am not interested in roles focused primarily on JavaScript frontend or mobile development.";

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

function ScoringPage() {
	const query = useScoringConfig();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="Scoring settings"
				subtitle="Filter out obvious non-fits, then let Claude score what's left for suitability."
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
	const [rubric, setRubric] = createSignal(props.data.suitabilityRubric ?? "");
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

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({
				suitabilityRubric: rubric(),
				notifyThreshold: threshold(),
				excludedTitleKeywords: splitList(titleKeywords()),
				excludedCompanies: splitList(companies()),
				excludedSeniority: seniority(),
				excludedLocations: splitList(locations()),
			});
			setSaved(true);
			setTimeout(() => setSaved(false), 3000);
		} catch {
			setSaveError("Failed to save. Please try again.");
		}
	};

	const handleUseTemplate = () => {
		setRubric(STARTER_RUBRIC);
	};

	return (
		<>
			<FormFeedback success={saved()} error={saveError()} />

			<div class="flex flex-col gap-5">
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<label for="rubric" class="text-base font-semibold text-foreground">
							Scoring rubric
						</label>
						<p class="mt-0.5 text-xs text-faint">
							Describe your ideal candidate profile. Claude uses this when
							scoring each job.
						</p>
					</div>
					<div class="px-5 py-4">
						<Show when={!rubric()}>
							<div class="mb-4 rounded-lg border border-border bg-surface-muted px-4 py-3 text-sm text-muted">
								<p class="font-medium text-foreground">No rubric set yet</p>
								<p class="mt-1 text-xs text-faint">
									Without a rubric, Claude has no profile to score against. Add
									one so jobs are ranked against your actual goals and
									experience.
								</p>
							</div>
						</Show>
						<textarea
							id="rubric"
							rows={7}
							value={rubric()}
							onInput={(e) => setRubric(e.currentTarget.value)}
							placeholder="Describe your background, preferred stack, work style, company size, location preferences…"
							class="w-full resize-y rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
						/>
						<Show when={!rubric()}>
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
