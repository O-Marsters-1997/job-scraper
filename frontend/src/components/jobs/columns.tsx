import { Link } from "@tanstack/solid-router";
import type { ColumnDef } from "@tanstack/solid-table";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { formatRelative } from "@/lib/datetime";
import { STATUS_FALLBACK_COLOUR } from "@/lib/status";
import { titleCase } from "@/lib/utils";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { JobActionsMenu } from "./JobActionsMenu";

export { titleCase };

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
					class="block max-w-[260px] truncate font-medium text-foreground transition-colors hover:text-primary"
					title={info.getValue() as string}
				>
					{info.getValue() as string}
				</Link>
			),
		},
		{
			accessorKey: "CompanySlug",
			header: "Company",
			cell: (info) => (
				<span
					class="block max-w-[180px] truncate text-muted"
					title={titleCase(info.getValue() as string)}
				>
					{titleCase(info.getValue() as string)}
				</span>
			),
		},
		{
			accessorKey: "Location",
			header: "Location",
			cell: (info) => {
				const val = info.getValue() as string | null | undefined;
				return val ? (
					<span class="block max-w-[160px] truncate text-muted" title={val}>
						{val}
					</span>
				) : (
					<span class="text-faint">—</span>
				);
			},
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
			accessorKey: "RelevanceScore",
			header: "Relevance",
			enableGlobalFilter: false,
			sortUndefined: -1,
			cell: (info) => {
				const val = info.getValue() as number | null | undefined;
				if (val === null || val === undefined) {
					return <span class="text-faint">—</span>;
				}
				return (
					<span class="font-mono text-xs tabular-nums text-muted">{val}</span>
				);
			},
		},
		{
			accessorKey: "SuitabilityScore",
			header: "Suitability",
			enableGlobalFilter: false,
			sortUndefined: -1,
			filterFn: "suitabilityMin" as unknown as "auto",
			cell: (info) => {
				const val = info.getValue() as number | null | undefined;
				if (val === null || val === undefined) {
					return <span class="text-faint">—</span>;
				}
				return (
					<span class="font-mono text-xs tabular-nums text-muted">{val}</span>
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
			cell: (info) => (
				<span class="font-mono text-xs tabular-nums text-faint">
					{formatRelative(info.getValue() as string)}
				</span>
			),
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
