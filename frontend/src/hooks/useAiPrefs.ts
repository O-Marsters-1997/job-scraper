import { createQuery, queryOptions } from "@tanstack/solid-query";
import { updateAiCredentials } from "../api/aiCredentials";
import { fetchAiPrefs } from "../api/aiPrefs";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const aiPrefsQueryOptions = queryOptions({
	queryKey: keys.aiPrefs,
	queryFn: fetchAiPrefs,
});

export function useAiPrefs() {
	return createQuery(() => aiPrefsQueryOptions);
}

export function useUpdateAiCredentials() {
	return useInvalidatingMutation(
		(payload: { provider: string; apiKey: string | null }) =>
			updateAiCredentials(payload),
		[keys.aiPrefs],
	);
}
