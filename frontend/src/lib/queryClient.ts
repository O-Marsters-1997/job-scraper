import { QueryClient } from "@tanstack/solid-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			retry: 1,
			staleTime: 30_000,
		},
	},
});
