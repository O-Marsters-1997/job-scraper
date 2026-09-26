import { createFileRoute, Link } from "@tanstack/solid-router";
import { For } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { Card } from "@/components/ui/card";
import { Switch, SwitchControl, SwitchThumb } from "@/components/ui/switch";
import { demoDataEnabled, setDemoData } from "@/lib/demoData";
import { SETTINGS_SECTIONS } from "@/lib/settingsSections";

export const Route = createFileRoute("/_auth/settings/")({
	component: SettingsHubPage,
});

function SettingsHubPage() {
	return (
		<div class="max-w-2xl px-7 py-6">
			<PageHeading title="Settings" />

			<Card class="mb-6 flex-row items-center justify-between gap-4 px-4 py-4">
				<div class="flex flex-col gap-0.5">
					<p class="text-sm font-medium text-foreground">Preview demo data</p>
					<p class="text-xs text-faint">
						Show seeded sample data for this session only.
					</p>
				</div>
				<Switch checked={demoDataEnabled()} onChange={setDemoData}>
					<SwitchControl>
						<SwitchThumb />
					</SwitchControl>
				</Switch>
			</Card>

			<div class="flex flex-col gap-6">
				<For each={SETTINGS_SECTIONS}>
					{(group) => (
						<div class="flex flex-col gap-2">
							<p class="px-1 text-2xs font-semibold uppercase tracking-wider text-faint">
								{group.heading}
							</p>
							<Card class="overflow-hidden divide-y divide-border">
								<For each={group.sections}>
									{(section) => (
										<Link
											to={section.to}
											class="flex items-center gap-4 px-4 py-4 transition-colors hover:bg-surface-muted"
										>
											<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-surface-muted text-muted">
												<section.icon />
											</div>
											<div class="flex flex-1 min-w-0 flex-col gap-0.5">
												<span class="text-sm font-medium text-foreground">
													{section.label}
												</span>
												<span class="text-xs text-faint">
													{section.description}
												</span>
											</div>
											<svg
												aria-hidden="true"
												class="shrink-0 text-faint"
												width="16"
												height="16"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												stroke-width="2"
												stroke-linecap="round"
												stroke-linejoin="round"
											>
												<polyline points="9 18 15 12 9 6" />
											</svg>
										</Link>
									)}
								</For>
							</Card>
						</div>
					)}
				</For>
			</div>
		</div>
	);
}
