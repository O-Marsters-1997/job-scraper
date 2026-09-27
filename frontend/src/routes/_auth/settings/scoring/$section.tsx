import { createFileRoute, redirect } from "@tanstack/solid-router";
import { SETTINGS_PAGES } from "@/lib/settingsSections";

export const Route = createFileRoute("/_auth/settings/scoring/$section")({
	beforeLoad: ({ params }) => {
		if (!SETTINGS_PAGES.includes(`/settings/scoring/${params.section}`)) {
			throw redirect({ to: "/settings/scoring" });
		}
	},
});
