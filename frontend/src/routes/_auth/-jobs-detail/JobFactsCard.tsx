import { Show } from "solid-js";
import { FactRow } from "@/components/FactRow";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import { titleCase } from "@/lib/utils";
import type { Job } from "@/types/job";
import { workStyle } from "./workStyle";

export function JobFactsCard(props: { job: Job }) {
	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Details</CardTitle>
			</CardHeader>
			<CardContent class="gap-2.5">
				<FactRow label="Company" value={titleCase(props.job.CompanySlug)} />
				<Show when={props.job.Location}>
					{(loc) => <FactRow label="Location" value={loc()} />}
				</Show>
				<Show when={workStyle(props.job.DaysInOffice)}>
					{(ws) => <FactRow label="Work style" value={ws()} />}
				</Show>
				<FactRow label="Source">
					<SourceBadge source={props.job.Source} />
				</FactRow>
				<FactRow label="Scraped">
					<span class="font-mono text-xs tabular-nums text-foreground">
						{formatDate(props.job.ScrapedAt)}
					</span>
				</FactRow>
				<Show when={props.job.EmploymentType}>
					{(et) => <FactRow label="Employment" value={et()} />}
				</Show>
				<Show when={props.job.SalaryRange}>
					{(sr) => (
						<FactRow label="Salary">
							<span class="font-mono text-xs tabular-nums text-foreground">
								{sr()}
							</span>
						</FactRow>
					)}
				</Show>
				<Show when={props.job.ExperienceLevel}>
					{(el) => <FactRow label="Experience" value={el()} />}
				</Show>
				<Show when={props.job.TeamName}>
					{(tn) => <FactRow label="Team" value={tn()} />}
				</Show>
				<Show when={props.job.CompanySize}>
					{(cs) => <FactRow label="Company size" value={cs()} />}
				</Show>
				<div class="pt-1">
					<Button
						as="a"
						href={props.job.URL}
						target="_blank"
						rel="noopener noreferrer"
						variant="secondary"
						size="sm"
						class="w-full"
					>
						View listing ↗
					</Button>
				</div>
			</CardContent>
		</Card>
	);
}
