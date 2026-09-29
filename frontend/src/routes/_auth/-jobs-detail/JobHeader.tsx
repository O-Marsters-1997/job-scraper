import { useNavigate } from "@tanstack/solid-router";
import type { JSX } from "solid-js";
import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { JobActionsMenu } from "@/components/jobs/JobActionsMenu";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { titleCase } from "@/lib/utils";
import type {
	ApplicationWithDetails,
	JobApplicationSummary,
} from "@/types/application";
import type { Job } from "@/types/job";
import { workStyle } from "./workStyle";

function MetaItem(props: { icon: JSX.Element; children: JSX.Element }) {
	return (
		<span class="flex items-center gap-1 text-xs text-muted">
			<span class="text-faint">{props.icon}</span>
			{props.children}
		</span>
	);
}

export function JobHeader(props: {
	job: Job;
	app: ApplicationWithDetails | undefined;
	appSummary: JobApplicationSummary | undefined;
	onTrack: () => void;
	onEdit: () => void;
}) {
	const navigate = useNavigate();

	return (
		<Card class="mb-4">
			<CardContent class="pt-5">
				<div class="flex gap-4">
					<div class="flex h-[52px] w-[52px] shrink-0 items-center justify-center rounded-xl bg-accent-subtle text-base font-semibold text-accent-text">
						{titleCase(props.job.CompanySlug).slice(0, 2)}
					</div>

					<div class="min-w-0 flex-1">
						<h1 class="text-xl font-bold tracking-tight text-foreground">
							{props.job.Title}
						</h1>
						<p class="mt-0.5 text-sm text-muted">
							{titleCase(props.job.CompanySlug)} · {props.job.Location}
						</p>

						<Show
							when={
								workStyle(props.job.DaysInOffice) ||
								props.job.EmploymentType ||
								props.job.SalaryRange ||
								props.job.ExperienceLevel
							}
						>
							<div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1">
								<Show when={workStyle(props.job.DaysInOffice)}>
									{(ws) => (
										<MetaItem icon={<Icon name="home" size={12} />}>
											{ws()}
										</MetaItem>
									)}
								</Show>
								<Show when={props.job.EmploymentType}>
									{(et) => (
										<MetaItem icon={<Icon name="briefcase" size={12} />}>
											{et()}
										</MetaItem>
									)}
								</Show>
								<Show when={props.job.SalaryRange}>
									{(sr) => (
										<MetaItem icon={<Icon name="dollar" size={12} />}>
											<span class="font-mono tabular-nums">{sr()}</span>
										</MetaItem>
									)}
								</Show>
								<Show when={props.job.ExperienceLevel}>
									{(el) => (
										<MetaItem icon={<Icon name="users" size={12} />}>
											{el()}
										</MetaItem>
									)}
								</Show>
							</div>
						</Show>

						<div class="mt-3 flex flex-wrap items-center gap-2">
							<Show when={props.app?.StatusName}>
								{(name) => (
									<StatusBadge
										name={name()}
										colour={props.app?.StatusColour || STATUS_FALLBACK_COLOUR}
									/>
								)}
							</Show>
							<SourceBadge source={props.job.Source} />
							<span class="font-mono text-xs tabular-nums text-faint">
								{formatDate(props.job.ScrapedAt)}
							</span>
						</div>
					</div>

					<div class="flex shrink-0 items-start gap-2">
						<Button
							size="sm"
							onClick={() =>
								navigate({
									to: "/jobs/$id/tailor",
									params: { id: props.job.ID },
								})
							}
						>
							Tailor CV
						</Button>
						<JobActionsMenu
							job={props.job}
							appSummary={props.appSummary}
							onTrack={props.onTrack}
							onEdit={props.onEdit}
						/>
					</div>
				</div>
			</CardContent>
		</Card>
	);
}
