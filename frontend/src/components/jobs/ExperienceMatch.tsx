import { Link } from "@tanstack/solid-router";
import { Match, Switch } from "solid-js";
import { Skeleton } from "@/components/ui/skeleton";
import { MissingAiKeyError } from "../../api/tailoring";
import { useExperienceMatch } from "../../hooks/useTailoring";

const EXPLANATION =
	"How well your best three Experience Bank bullets fit this job. Separate from Suitability, which scores the job against your preferences.";

export function ExperienceMatch(props: { jobId: string }) {
	const match = useExperienceMatch(() => props.jobId);
	return (
		<div class="space-y-1" title={EXPLANATION}>
			<p class="text-xs font-medium text-muted">Experience match</p>
			<Switch>
				<Match when={match.isPending}>
					<Skeleton class="h-6 w-16" />
				</Match>
				<Match when={match.error instanceof MissingAiKeyError}>
					<p class="text-xs text-faint">
						Needs an OpenRouter key.{" "}
						<Link to="/settings/ai" class="text-accent-text underline">
							Add one in Settings, AI
						</Link>
					</p>
				</Match>
				<Match when={match.isError}>
					<p class="text-xs text-faint">Could not compute.</p>
				</Match>
				<Match when={match.data?.score == null}>
					<p class="text-xs text-faint">Add experience to see a match.</p>
				</Match>
				<Match when={match.data}>
					{(data) => (
						<span class="font-mono text-lg font-semibold tabular-nums text-foreground">
							{Math.round(((data().score ?? 0) + 1) * 50)}%
						</span>
					)}
				</Match>
			</Switch>
		</div>
	);
}
