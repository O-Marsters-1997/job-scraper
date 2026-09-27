import { createFileRoute } from "@tanstack/solid-router";
import { SettingsLayout } from "@/components/SettingsLayout";

export const Route = createFileRoute("/_auth/settings")({
	component: SettingsLayout,
});
