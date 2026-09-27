import { createFileRoute, redirect } from "@tanstack/solid-router";

export const Route = createFileRoute("/_auth/settings/scoring/")({
	beforeLoad: () => {
		throw redirect({
			to: "/settings/scoring/$section",
			params: { section: "role" },
		});
	},
});
