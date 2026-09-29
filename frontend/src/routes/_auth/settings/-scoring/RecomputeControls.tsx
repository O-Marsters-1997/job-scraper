import { Show } from "solid-js";
import { Button } from "@/components/ui/button";
import {
	useRecomputeScores,
	useScoringStatus,
} from "../../../../hooks/useScores";

export function RecomputeControls(props: {
	onStart: () => void;
	onDone: (message: string) => void;
	onError: (message: string) => void;
}) {
	const recompute = useRecomputeScores();
	const scoringStatus = useScoringStatus();

	const handleRecompute = async () => {
		props.onStart();
		try {
			const result = await recompute.mutateAsync();
			props.onDone(`${result.recomputed} jobs re-ranked.`);
		} catch {
			props.onError("Could not recompute scores. Try again.");
		}
	};

	return (
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
		</>
	);
}
