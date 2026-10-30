import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { appendJobFeedback } from "../api/scoreFeedback";
import { fetchPlaygroundQueue } from "../api/scoringPlayground";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function usePlaygroundVerdict() {
	return useInvalidatingMutation(appendJobFeedback, [
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
