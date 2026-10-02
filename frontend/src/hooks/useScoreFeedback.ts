import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	appendOverallFeedback,
	fetchScoreFeedback,
} from "../api/scoreFeedback";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

const scoreFeedbackQueryOptions = queryOptions({
	queryKey: keys.scoreFeedback,
	queryFn: fetchScoreFeedback,
});

export function useScoreFeedback() {
	return createQuery(() => scoreFeedbackQueryOptions);
}

export function useAppendOverallFeedback() {
	return useInvalidatingMutation(appendOverallFeedback, [keys.scoreFeedback]);
}
