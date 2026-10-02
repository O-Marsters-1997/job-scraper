import { createRootRoute, Outlet } from "@tanstack/solid-router";
import { lazy, Show } from "solid-js";

import "../styles.css";

// Dev-only: the dynamic import is gated on import.meta.env.DEV so Vite drops the
// devtools chunk from the production bundle entirely.
const TanStackRouterDevtools = import.meta.env.DEV
	? lazy(() =>
			import("@tanstack/solid-router-devtools").then((m) => ({
				default: m.TanStackRouterDevtools,
			})),
		)
	: () => null;

const ScoreFeedback =
	import.meta.env.VITE_SCORING_FEEDBACK === "true"
		? lazy(() => import("@/components/score-feedback"))
		: () => null;

export const Route = createRootRoute({
	component: RootComponent,
});

function RootComponent() {
	return (
		<>
			<Outlet />
			<ScoreFeedback />
			<Show when={import.meta.env.DEV}>
				<TanStackRouterDevtools position="bottom-right" />
			</Show>
		</>
	);
}
