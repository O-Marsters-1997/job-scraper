import {
	createMutation,
	createQuery,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchScoringStatus, queueRescore } from "../api/scores";

export function useScoringStatus() {
	return createQuery(() => ({
		queryKey: ["scores", "status"],
		queryFn: fetchScoringStatus,
	}));
}

export function useQueueRescore() {
	const client = useQueryClient();
	return createMutation(() => ({
		mutationFn: queueRescore,
		onSuccess: () =>
			client.invalidateQueries({ queryKey: ["scores", "status"] }),
	}));
}
