import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { SuitabilityScoreValue } from "@/components/jobs/SuitabilityScoreValue";
import { SourceBadge } from "@/components/SourceBadge";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { formatDate } from "@/lib/datetime";
import { titleCase } from "@/lib/utils";
import type { Job } from "@/types/job";

export function RecentJobsCard(props: { jobs: Job[]; ranked: boolean }) {
	return (
		<Card>
			<div class="flex items-center justify-between border-b border-border px-5 py-4">
				<h2 class="text-base font-semibold leading-snug text-foreground">
					{props.ranked ? "Recent high-value jobs" : "Newest unseen jobs"}
				</h2>
				<Link
					to="/jobs"
					search={{ scored: props.ranked, seen: "unseen" }}
					class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
				>
					View all →
				</Link>
			</div>
			<Show when={!props.ranked}>
				<p class="border-b border-border px-5 py-2 text-xs text-faint">
					Not ranked: AI scoring is off.
				</p>
			</Show>
			<Show
				when={props.jobs.length > 0}
				fallback={
					<p class="px-5 py-4 text-sm text-faint">
						{props.ranked
							? "No recent unseen Good or better jobs."
							: "No unseen jobs."}
					</p>
				}
			>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Role</TableHead>
							<TableHead>Company</TableHead>
							<TableHead>Source</TableHead>
							<Show when={props.ranked}>
								<TableHead>Match</TableHead>
							</Show>
							<TableHead>Scraped</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						<For each={props.jobs}>
							{(job) => (
								<TableRow>
									<TableCell class="max-w-[220px] truncate font-medium text-foreground">
										{job.Title}
									</TableCell>
									<TableCell
										class="max-w-[160px] truncate text-muted"
										title={titleCase(job.CompanySlug)}
									>
										{titleCase(job.CompanySlug)}
									</TableCell>
									<TableCell>
										<SourceBadge source={job.Source} />
									</TableCell>
									<Show when={props.ranked}>
										<TableCell>
											<SuitabilityScoreValue
												score={job.SuitabilityScore}
												band={job.Band}
											/>
										</TableCell>
									</Show>
									<TableCell class="font-mono text-xs tabular-nums text-faint">
										{formatDate(job.ScrapedAt)}
									</TableCell>
								</TableRow>
							)}
						</For>
					</TableBody>
				</Table>
			</Show>
		</Card>
	);
}
