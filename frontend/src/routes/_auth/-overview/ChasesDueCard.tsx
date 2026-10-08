import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { StatusBadge } from "@/components/StatusBadge";
import { Card } from "@/components/ui/card";
import { chaseOverdueDays, formatChaseDate } from "@/lib/chase";
import { titleCase } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";

function overdueLabel(days: number) {
	return `overdue by ${days} ${days === 1 ? "day" : "days"}`;
}

export function ChasesDueCard(props: { chases: ApplicationWithDetails[] }) {
	const now = new Date();
	return (
		<Card class="mb-3">
			<div class="flex items-center justify-between border-b border-border px-5 py-4">
				<h2 class="text-base font-semibold leading-snug text-foreground">
					Chases due
				</h2>
				<Link
					to="/applications"
					search={{ status: undefined, chase: true }}
					class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
				>
					View chases →
				</Link>
			</div>
			<Show
				when={props.chases.length > 0}
				fallback={<p class="px-5 py-4 text-sm text-faint">No chases due</p>}
			>
				<For each={props.chases}>
					{(app) => {
						const days = chaseOverdueDays(app.ChaseBy!, now);
						return (
							<div class="flex items-center gap-3 border-b border-border px-4 py-3 last:border-0">
								<div class="min-w-0 flex-1">
									<p class="truncate text-sm font-medium text-foreground">
										{app.JobTitle}
									</p>
									<p class="text-xs text-faint">
										{titleCase(app.JobCompanySlug)}
									</p>
								</div>
								<div class="text-right text-xs">
									<p class="font-mono tabular-nums text-foreground">
										{formatChaseDate(app.ChaseBy!)}
									</p>
									<Show when={days > 0}>
										<p class="text-destructive-strong">{overdueLabel(days)}</p>
									</Show>
								</div>
								<Show when={app.StatusName}>
									<StatusBadge
										name={app.StatusName}
										colour={app.StatusColour}
									/>
								</Show>
							</div>
						);
					}}
				</For>
			</Show>
		</Card>
	);
}
