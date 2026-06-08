import { useLocation } from "@tanstack/solid-router";
import SettingsPanel from "./SettingsPanel";

const PAGE_LABELS: Record<string, string> = {
	"/overview": "Overview",
	"/jobs": "Jobs",
	"/applications": "Applications",
	"/settings/statuses": "Statuses",
};

export default function Topbar() {
	const location = useLocation();
	const page = () => PAGE_LABELS[location().pathname] ?? "Job Scraper";

	return (
		<header class="flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface px-6">
			<nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-sm">
				<span class="text-faint">Job Scraper</span>
				<span class="text-border-strong">/</span>
				<span class="font-semibold text-foreground">{page()}</span>
			</nav>
			<SettingsPanel />
		</header>
	);
}
