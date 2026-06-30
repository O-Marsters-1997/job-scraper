import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import type { ScoringConfig } from "../../../api/scoringConfig";
import {
	useScoringConfig,
	useUpdateScoringConfig,
} from "../../../hooks/useScoringConfig";

export const Route = createFileRoute("/_auth/settings/scoring")({
	component: ScoringPage,
});

const STARTER_RUBRIC =
	"I am a software engineer with 3+ years of experience in backend development. I am looking for roles that involve: Go or Python, distributed systems or APIs, remote or hybrid work. I prefer companies with fewer than 500 employees. I am not interested in roles focused primarily on JavaScript frontend or mobile development.";

function ScoringPage() {
	const query = useScoringConfig();
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading
				title="Scoring settings"
				subtitle="Control how Claude scores job listings for suitability and when you receive notifications."
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
	const [rubric, setRubric] = createSignal(props.data.suitabilityRubric ?? "");
	const [cutoff, setCutoff] = createSignal(props.data.relevanceCutoff ?? 0);
	const [threshold, setThreshold] = createSignal(
		props.data.notifyThreshold ?? 70,
	);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await mutation.mutateAsync({
				suitabilityRubric: rubric(),
				relevanceCutoff: cutoff(),
				notifyThreshold: threshold(),
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
				{/* Rubric */}
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

				{/* Cutoff + threshold */}
				<Card class="overflow-hidden">
					<div class="border-b border-border px-5 py-4">
						<p class="text-base font-semibold text-foreground">Thresholds</p>
						<p class="mt-0.5 text-xs text-faint">
							Tune when AI scoring runs and when notifications are sent.
						</p>
					</div>
					<div class="divide-y divide-border">
						<div class="px-5 py-4">
							<label for="cutoff" class="text-xs font-medium text-foreground">
								Relevance cutoff
							</label>
							<p class="mt-0.5 text-xs text-faint">
								Jobs scoring below this on keyword/location matching skip AI
								scoring entirely.
							</p>
							<div class="mt-2 flex items-center gap-2">
								<Input
									id="cutoff"
									type="number"
									min="0"
									max="100"
									value={cutoff()}
									onInput={(e) => setCutoff(Number(e.currentTarget.value))}
									class="w-20"
								/>
								<span class="text-xs text-faint">out of 100</span>
							</div>
						</div>

						<div class="px-5 py-4">
							<label
								for="threshold"
								class="text-xs font-medium text-foreground"
							>
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
					</div>
				</Card>

				<div class="flex items-center gap-3">
					<Button onClick={handleSave} disabled={mutation.isPending}>
						{mutation.isPending ? "Saving…" : "Save"}
					</Button>
				</div>

				<p class="text-xs text-faint">
					Changes apply to future scoring only. Already-scored jobs are not
					re-scored.
				</p>
			</div>
		</>
	);
}
