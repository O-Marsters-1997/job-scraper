import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { resolveBoard } from "../api/sources";

export function useResolveBoard(url: () => string) {
	return createQuery(() => ({
		queryKey: [...keys.sources, "resolve", url()],
		queryFn: () => resolveBoard(url()),
		enabled: /^https?:\/\/\S+$/i.test(url()),
		staleTime: Infinity,
		retry: false,
	}));
}
