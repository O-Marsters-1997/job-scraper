import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
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

export function RecentJobsCard(props: { jobs: Job[] }) {
	return (
		<Card>
			<div class="flex items-center justify-between border-b border-border px-5 py-4">
				<h2 class="text-base font-semibold leading-snug text-foreground">
					Recent jobs
				</h2>
				<Link
					to="/jobs"
					class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
				>
					View all →
				</Link>
			</div>
			<Show
				when={props.jobs.length > 0}
				fallback={
					<p class="px-5 py-4 text-sm text-faint">No jobs scraped yet.</p>
				}
			>
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Role</TableHead>
							<TableHead>Company</TableHead>
							<TableHead>Source</TableHead>
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
