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
			id: "expand",
			enableSorting: false,
			enableGlobalFilter: false,
			header: () => <span class="sr-only">Expand</span>,
			cell: (info) => {
				const rowId = info.row.original.ID;
				const meta = info.table.options.meta;
				const expanded = () => meta?.isExpanded(rowId) ?? false;
				return (
					<button
						type="button"
						aria-label={expanded() ? "Collapse row" : "Expand row"}
						aria-expanded={expanded()}
						onClick={(e) => {
							e.stopPropagation();
							meta?.toggleExpanded(rowId);
						}}
						class="flex size-6 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-accent-text"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="14"
							height="14"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
							style={{
								transform: expanded() ? "rotate(180deg)" : "rotate(0deg)",
								transition: "transform 150ms ease",
							}}
						>
							<polyline points="6 9 12 15 18 9" />
						</svg>
					</button>
				);
			},
		},
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
