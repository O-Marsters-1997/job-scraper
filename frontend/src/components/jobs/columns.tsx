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
import { ScoreCircle } from "./ScoreCircle";

export interface JobTableContext {
	appsForJobs: () => Record<string, JobApplicationSummary> | undefined;
	onTrack: (jobId: string) => void;
	onEdit: (jobId: string) => void;
}

// Extend TanStack Table's meta type so cells can read expand state
declare module "@tanstack/solid-table" {
	// biome-ignore lint/suspicious/noExplicitAny: table meta extension
	interface TableMeta<TData> {
		isExpanded: (rowId: string) => boolean;
		toggleExpanded: (rowId: string) => void;
	}
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
			accessorKey: "SuitabilityScore",
			header: "Suitability",
			enableGlobalFilter: false,
			sortUndefined: -1,
			cell: (info) => {
				const val = info.getValue() as number | null | undefined;
				const rowId = info.row.original.ID;
				const meta = info.table.options.meta;
				const expanded = () => meta?.isExpanded(rowId) ?? false;
				return (
					<button
						type="button"
						aria-label={expanded() ? "Collapse details" : "Expand details"}
						aria-expanded={expanded()}
						onClick={(e) => {
							e.stopPropagation();
							meta?.toggleExpanded(rowId);
						}}
						class={`rounded-full transition-all hover:opacity-80${expanded() ? " ring-2 ring-primary ring-offset-1" : ""}`}
					>
						{val != null ? (
							<ScoreCircle value={val} />
						) : (
							<span class="inline-flex size-7 items-center justify-center text-faint">
								—
							</span>
						)}
					</button>
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
