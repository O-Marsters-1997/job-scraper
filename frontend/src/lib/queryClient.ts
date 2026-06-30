import { QueryClient } from "@tanstack/solid-query";

export const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			// Retry once before showing the error UI — absorbs transient blips.
			retry: 1,
		},
	},
});
