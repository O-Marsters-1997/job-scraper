import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchAiPrefs, updateAiPrefs } from "../api/aiPrefs";

export const aiPrefsQueryOptions = queryOptions({
	queryKey: ["ai-prefs"],
	queryFn: fetchAiPrefs,
});

export function useAiPrefs() {
	return createQuery(() => aiPrefsQueryOptions);
}

export function useUpdateAiPrefs() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: { suitabilityModel: string }) =>
			updateAiPrefs(payload),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ai-prefs"] }),
	}));
}
