import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { fetchPlaygroundQueue } from "../api/scoringPlayground";

export function usePlaygroundQueue() {
	return createQuery(() => ({
		queryKey: keys.playgroundQueue,
		queryFn: fetchPlaygroundQueue,
		refetchOnWindowFocus: false,
	}));
}
