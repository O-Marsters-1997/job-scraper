import { createFileRoute, redirect } from "@tanstack/solid-router";

export const Route = createFileRoute("/_auth/settings/")({
	beforeLoad: () => {
		throw redirect({ to: "/settings/profile" });
	},
});
