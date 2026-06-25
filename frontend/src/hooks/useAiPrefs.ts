import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { fetchAiPrefs, updateAiPrefs } from "../api/aiPrefs";
import { updateAiCredentials } from "../api/aiCredentials";

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

export function useUpdateAiCredentials() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: { provider: string; apiKey: string | null }) =>
			updateAiCredentials(payload),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: ["ai-prefs"] }),
	}));
}
