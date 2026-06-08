import type { ColumnDef } from "@tanstack/solid-table";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/StatusBadge";
import { JobActionsMenu } from "./JobActionsMenu";
import type { Job } from "@/types/job";
import type { JobApplicationSummary } from "@/types/application";

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
				<span class="font-medium text-foreground">
					{info.getValue() as string}
				</span>
			),
		},
		{
			accessorKey: "CompanySlug",
			header: "Company",
			cell: (info) => (
				<span class="text-muted">{info.getValue() as string}</span>
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
			accessorKey: "Source",
			header: "Source",
			cell: (info) => (
				<Badge variant="source">{info.getValue() as string}</Badge>
			),
		},
		{
			accessorKey: "ScrapedAt",
			header: "Scraped",
			enableGlobalFilter: false,
			cell: (info) => {
				const raw = info.getValue() as string;
				return (
					<span class="font-mono text-xs tabular-nums text-faint">
						{new Date(raw).toLocaleDateString("en-GB", {
							day: "numeric",
							month: "short",
							year: "numeric",
						})}
					</span>
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
						colour={summary.StatusColour || "#64748b"}
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
