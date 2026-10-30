import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { appendJobFeedback, deleteScoreFeedback } from "../api/scoreFeedback";
import { fetchPlaygroundQueue } from "../api/scoringPlayground";
import type { JobFeedbackInput } from "../types/scoreFeedback";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

async function replaceJobFeedback({
	replaces,
	...input
}: JobFeedbackInput & { replaces?: string | undefined }) {
	if (replaces) await deleteScoreFeedback(replaces);
	return appendJobFeedback(input);
}

export function usePlaygroundVerdict() {
	return useInvalidatingMutation(replaceJobFeedback, [
		keys.scoreFeedback,
		keys.playgroundQueue,
	]);
}

export function usePlaygroundQueue() {
	return createQuery(() => ({
		queryKey: keys.playgroundQueue,
		queryFn: fetchPlaygroundQueue,
		refetchOnWindowFocus: false,
	}));
}
