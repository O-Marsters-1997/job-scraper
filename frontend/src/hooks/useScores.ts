import {
	createMutation,
	createQuery,
	useQueryClient,
} from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringStatus, recomputeScores } from "../api/scores";

export function useScoringStatus() {
	return createQuery(() => ({
		queryKey: keys.scores.status(),
		queryFn: fetchScoringStatus,
	}));
}

export function useRecomputeScores() {
	const client = useQueryClient();
	return createMutation(() => ({
		mutationFn: recomputeScores,
		onSuccess: () => {
			client.invalidateQueries({ queryKey: keys.jobs.all });
			client.invalidateQueries({ queryKey: keys.scores.all });
		},
	}));
}
