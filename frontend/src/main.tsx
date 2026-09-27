import { QueryClientProvider } from "@tanstack/solid-query";
import { createRouter, RouterProvider } from "@tanstack/solid-router";
import { render } from "solid-js/web";
import { ErrorState, NotFoundState } from "./components/ErrorState";
import { queryClient } from "./lib/queryClient";
import { loadTweaks } from "./lib/tweaks";
import { applyTweaks } from "./lib/tweaks.apply";
import { routeTree } from "./routeTree.gen";

applyTweaks(loadTweaks());

const router = createRouter({
	routeTree,
	defaultPreload: "intent",
	defaultPreloadStaleTime: 0,
	scrollRestoration: true,
	defaultErrorComponent: (props) => (
		<ErrorState error={props.error} onRetry={props.reset} />
	),
	defaultNotFoundComponent: () => <NotFoundState />,
});

declare module "@tanstack/solid-router" {
	interface Register {
		router: typeof router;
	}
}

const rootElement = document.getElementById("app");

if (rootElement && !rootElement.innerHTML) {
	render(
		() => (
			<QueryClientProvider client={queryClient}>
				<RouterProvider router={router} />
			</QueryClientProvider>
		),
		rootElement,
	);
}
