import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import { updateAiCredentials } from "../api/aiCredentials";
import { fetchAiPrefs } from "../api/aiPrefs";
import { keys } from "../api/keys";

export const aiPrefsQueryOptions = queryOptions({
	queryKey: keys.aiPrefs,
	queryFn: fetchAiPrefs,
});

export function useAiPrefs() {
	return createQuery(() => aiPrefsQueryOptions);
}

export function useUpdateAiCredentials() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: { provider: string; apiKey: string | null }) =>
			updateAiCredentials(payload),
		onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.aiPrefs }),
	}));
}
