import { createFileRoute } from "@tanstack/solid-router";
import { createEffect, createSignal, Show } from "solid-js";
import { SkeletonList } from "@/components/ui/skeleton";
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
	const saveMutation = useUpdateScoringConfig();

	const [rubric, setRubric] = createSignal("");
	const [cutoff, setCutoff] = createSignal(0);
	const [threshold, setThreshold] = createSignal(70);
	const [saved, setSaved] = createSignal(false);
	const [saveError, setSaveError] = createSignal<string | null>(null);

	createEffect(() => {
		const data = query.data;
		if (!data) return;
		setRubric(data.suitabilityRubric ?? "");
		setCutoff(data.relevanceCutoff ?? 0);
		setThreshold(data.notifyThreshold ?? 70);
	});

	const handleSave = async () => {
		setSaved(false);
		setSaveError(null);
		try {
			await saveMutation.mutateAsync({
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
		<div class="max-w-2xl px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Scoring settings
				</h1>
				<p class="mt-0.5 text-xs text-faint">
					Control how Claude scores job listings for suitability and when you
					receive notifications.
				</p>
			</div>

			<Show when={saved()}>
				<div class="mb-4 rounded-lg border border-primary/30 bg-accent-subtle px-4 py-3 text-sm text-primary">
					Saved.
				</div>
			</Show>

			<Show when={saveError()}>
				<div class="mb-4 rounded-lg border border-destructive/30 bg-destructive-subtle px-4 py-3 text-sm text-destructive-strong">
					{saveError()}
				</div>
			</Show>

			<Show when={query.isPending}>
				<SkeletonList rows={4} />
			</Show>

			<Show when={query.isSuccess}>
				<div class="flex flex-col gap-5">
					{/* Rubric */}
					<div class="overflow-hidden rounded-xl border border-border bg-surface">
						<div class="border-b border-border px-5 py-4">
							<p class="text-base font-semibold text-foreground">
								Scoring rubric
							</p>
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
										Without a rubric, Claude has no profile to score against.
										Add one so jobs are ranked against your actual goals and
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
								class="w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10 resize-y"
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
					</div>

					{/* Cutoff + threshold */}
					<div class="overflow-hidden rounded-xl border border-border bg-surface">
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
									<input
										id="cutoff"
										type="number"
										min="0"
										max="100"
										value={cutoff()}
										onInput={(e) => setCutoff(Number(e.currentTarget.value))}
										class="w-20 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
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
									You only receive notifications for jobs scoring at or above
									this suitability score.
								</p>
								<div class="mt-2 flex items-center gap-2">
									<input
										id="threshold"
										type="number"
										min="0"
										max="100"
										value={threshold()}
										onInput={(e) => setThreshold(Number(e.currentTarget.value))}
										class="w-20 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
									/>
									<span class="text-xs text-faint">out of 100</span>
								</div>
							</div>
						</div>
					</div>

					<div class="flex items-center gap-3">
						<button
							type="button"
							onClick={handleSave}
							disabled={saveMutation.isPending}
							class="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover disabled:opacity-50"
						>
							{saveMutation.isPending ? "Saving…" : "Save"}
						</button>
					</div>

					<p class="text-xs text-faint">
						Changes apply to future scoring only. Already-scored jobs are not
						re-scored.
					</p>
				</div>
			</Show>
		</div>
	);
}
