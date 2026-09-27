import {
	createMutation,
	createQuery,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchScoringStatus, recomputeScores } from "../api/scores";

export function useScoringStatus() {
	return createQuery(() => ({
		queryKey: ["scores", "status"],
		queryFn: fetchScoringStatus,
	}));
}

export function useRecomputeScores() {
	const client = useQueryClient();
	return createMutation(() => ({
		mutationFn: recomputeScores,
		onSuccess: () => client.invalidateQueries({ queryKey: ["jobs"] }),
	}));
}
