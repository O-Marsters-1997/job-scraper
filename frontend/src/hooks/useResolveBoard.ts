import { createQuery } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import { ResolveError, resolveUrl } from "../api/sources";

export function useResolveBoard(url: () => string) {
	return createQuery(() => ({
		queryKey: [...keys.sources, "resolve", url()],
		queryFn: async () => {
			try {
				return await resolveUrl(url());
			} catch (err) {
				if (err instanceof ResolveError) return null;
				throw err;
			}
		},
		enabled: /^https?:\/\/\S+$/i.test(url()),
		staleTime: Infinity,
		retry: false,
	}));
}
