import { useLocation } from "@tanstack/solid-router";

const PAGE_LABELS: Record<string, string> = {
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
			<div
				class="flex h-8 w-8 items-center justify-center rounded-full border border-accent-border bg-accent-subtle text-accent-text"
				aria-hidden="true"
			>
				<svg
					width="15"
					height="15"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
					<circle cx="12" cy="7" r="4" />
				</svg>
			</div>
		</header>
	);
}
