import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { StatusBadge } from "@/components/StatusBadge";
import { Card } from "@/components/ui/card";
import { titleCase } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";

export function RecentApplicationsCard(props: {
	applications: ApplicationWithDetails[];
}) {
	return (
		<Card>
			<div class="flex items-center justify-between border-b border-border px-5 py-4">
				<h2 class="text-base font-semibold leading-snug text-foreground">
					Recent applications
				</h2>
				<Link
					to="/applications"
					search={{ status: undefined, chase: undefined }}
					class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
				>
					View all →
				</Link>
			</div>
			<Show
				when={props.applications.length > 0}
				fallback={
					<p class="px-5 py-4 text-sm text-faint">No applications yet.</p>
				}
			>
				<For each={props.applications}>
					{(app) => (
						<div class="flex items-center gap-3 border-b border-border px-4 py-3 last:border-0">
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium text-foreground">
									{app.JobTitle}
								</p>
								<p class="text-xs text-faint">
									{titleCase(app.JobCompanySlug)}
									{app.JobLocation ? ` · ${app.JobLocation}` : ""}
								</p>
							</div>
							<Show when={app.StatusName}>
								<StatusBadge name={app.StatusName} colour={app.StatusColour} />
							</Show>
						</div>
					)}
				</For>
			</Show>
		</Card>
	);
}
