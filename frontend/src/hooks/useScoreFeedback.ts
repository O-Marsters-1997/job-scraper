import { createQuery, keepPreviousData } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	appendJobFeedback,
	appendOverallFeedback,
	deleteScoreFeedback,
	fetchScoreFeedback,
} from "../api/scoreFeedback";
import type { ScoreFeedbackKind } from "../types/scoreFeedback";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useScoreFeedback(
	kind: () => ScoreFeedbackKind | undefined,
	page: () => number,
) {
	return createQuery(() => ({
		queryKey: [...keys.scoreFeedback, kind() ?? "all", page()],
		queryFn: () => fetchScoreFeedback(kind(), page()),
		placeholderData: keepPreviousData,
	}));
}

export function useDeleteScoreFeedback() {
	return useInvalidatingMutation(deleteScoreFeedback, [keys.scoreFeedback]);
}

export function useAppendOverallFeedback() {
	return useInvalidatingMutation(appendOverallFeedback, [keys.scoreFeedback]);
}

export function useAppendJobFeedback() {
	return useInvalidatingMutation(appendJobFeedback, [keys.scoreFeedback]);
}
