import { Link } from "@tanstack/solid-router";
import type { ColumnDef } from "@tanstack/solid-table";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { formatDate, formatRelative } from "@/lib/datetime";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { JobActionsMenu } from "./JobActionsMenu";

export function titleCase(slug: string): string {
	return slug
		.split(/[-_\s]+/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(" ");
}

export interface JobTableContext {
	appsForJobs: () => Record<string, JobApplicationSummary> | undefined;
	onTrack: (jobId: string) => void;
	onEdit: (jobId: string) => void;
}

export function createJobColumns(
	ctx: JobTableContext,
): ColumnDef<Job, unknown>[] {
	return [
		{
			accessorKey: "Title",
			header: "Title",
			cell: (info) => (
				<Link
					to="/jobs/$id"
					params={{ id: info.row.original.ID }}
					class="font-medium text-foreground transition-colors hover:text-primary"
				>
					{info.getValue() as string}
				</Link>
			),
		},
		{
			accessorKey: "CompanySlug",
			header: "Company",
			cell: (info) => (
				<span class="text-muted">{titleCase(info.getValue() as string)}</span>
			),
		},
		{
			accessorKey: "Location",
			header: "Location",
			cell: (info) => (
				<span class="text-muted">{info.getValue() as string}</span>
			),
		},
		{
			accessorKey: "DaysInOffice",
			header: "Office",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => {
				const val = info.getValue() as number | null | undefined;
				if (val === null || val === undefined) {
					return <span class="text-faint">—</span>;
				}
				if (val === 0) return <span class="text-muted">Remote</span>;
				return (
					<span class="font-mono text-xs tabular-nums text-muted">
						{val}d/wk
					</span>
				);
			},
		},
		{
			accessorKey: "Source",
			header: "Source",
			cell: (info) => <SourceBadge source={info.getValue() as string} />,
		},
		{
			accessorKey: "ScrapedAt",
			header: "Scraped",
			enableGlobalFilter: false,
			cell: (info) => {
				const raw = info.getValue() as string;
				return (
					<div class="flex flex-col gap-0.5">
						<span class="font-mono text-xs tabular-nums text-foreground">
							{formatDate(raw)}
						</span>
						<span class="text-[10px] text-faint">{formatRelative(raw)}</span>
					</div>
				);
			},
		},
		{
			id: "status",
			header: "Status",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => {
				const summary = ctx.appsForJobs()?.[info.row.original.ID];
				return summary?.StatusName ? (
					<StatusBadge
						name={summary.StatusName}
						colour={summary.StatusColour || STATUS_FALLBACK_COLOUR}
					/>
				) : (
					<span class="text-faint">—</span>
				);
			},
		},
		{
			id: "actions",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => (
				<div class="flex justify-end">
					<JobActionsMenu
						job={info.row.original}
						appSummary={ctx.appsForJobs()?.[info.row.original.ID]}
						onTrack={() => ctx.onTrack(info.row.original.ID)}
						onEdit={() => ctx.onEdit(info.row.original.ID)}
					/>
				</div>
			),
		},
	];
}
