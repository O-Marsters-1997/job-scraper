import { Show } from "solid-js";
import { FactRow } from "@/components/FactRow";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import type { ApplicationWithDetails } from "@/types/application";
import { ChaseRow } from "./ChaseRow";

export function ApplicationCard(props: {
	app: ApplicationWithDetails | undefined;
	onTrack: () => void;
}) {
	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Application</CardTitle>
			</CardHeader>
			<Show
				when={props.app}
				fallback={
					<CardContent class="gap-3">
						<p class="text-xs text-faint">Not yet tracked</p>
						<Button class="w-full" onClick={props.onTrack}>
							Track application
						</Button>
					</CardContent>
				}
			>
				{(a) => (
					<CardContent class="gap-2.5">
						<div>
							<StatusBadge name={a().StatusName} colour={a().StatusColour} />
						</div>
						<Show when={a().AppliedAt}>
							{(date) => (
								<FactRow label="Applied">
									<span class="font-mono text-xs tabular-nums text-foreground">
										{formatDate(date())}
									</span>
								</FactRow>
							)}
						</Show>
						<Show when={a().SalaryInfo}>
							{(salary) => (
								<FactRow label="Salary">
									<span class="font-mono text-xs tabular-nums text-foreground">
										{salary()}
									</span>
								</FactRow>
							)}
						</Show>
						<Show when={a().Notes}>
							{(notes) => (
								<p class="rounded-md bg-surface-muted px-3 py-2 text-xs text-muted">
									{notes()}
								</p>
							)}
						</Show>
						<ChaseRow
							applicationId={a().ID}
							statusId={a().StatusID}
							chaseBy={a().ChaseBy}
						/>
						<div class="pt-0.5">
							<Button
								variant="outline"
								size="sm"
								class="w-full"
								onClick={props.onTrack}
							>
								Edit
							</Button>
						</div>
					</CardContent>
				)}
			</Show>
		</Card>
	);
}
