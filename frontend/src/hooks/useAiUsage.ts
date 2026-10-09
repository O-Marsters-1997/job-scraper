import { createQuery, queryOptions } from "@tanstack/solid-query";
import { fetchAiUsage } from "../api/aiUsage";
import { keys } from "../api/keys";

export const aiUsageQueryOptions = queryOptions({
	queryKey: keys.aiUsage,
	queryFn: fetchAiUsage,
	staleTime: 5 * 60 * 1000,
});

export function useAiUsage() {
	return createQuery(() => aiUsageQueryOptions);
}
