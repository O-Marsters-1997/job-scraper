import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchScoringStatus, recomputeScores } from "../api/scores";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useScoringStatus() {
	return createQuery(() => ({
		queryKey: keys.scores.status(),
		queryFn: fetchScoringStatus,
	}));
}

export function useRecomputeScores() {
	return useInvalidatingMutation(recomputeScores, [
		keys.jobs.all,
		keys.scores.all,
	]);
}
