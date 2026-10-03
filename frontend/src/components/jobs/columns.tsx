import { Link } from "@tanstack/solid-router";
import type { ColumnDef } from "@tanstack/solid-table";
import { Match, Show, Switch } from "solid-js";
import { Icon } from "@/components/Icon";
import { SourceBadge } from "@/components/SourceBadge";
import { StatusBadge } from "@/components/StatusBadge";
import { Badge } from "@/components/ui/badge";
import { formatRelative } from "@/lib/datetime";
import { titleCase } from "@/lib/utils";
import type { JobApplicationSummary } from "@/types/application";
import type { Job } from "@/types/job";
import { JobActionsMenu } from "./JobActionsMenu";
import { SuitabilityScoreValue } from "./SuitabilityScoreValue";

export interface JobTableContext {
	appsForJobs: () => Record<string, JobApplicationSummary> | undefined;
	onTrack: (jobId: string) => void;
	onDismiss: (job: Job) => void;
}

// Extend TanStack Table's meta type so cells can read expand state
declare module "@tanstack/solid-table" {
	interface TableMeta<TData> {
		isExpanded: (rowId: string) => boolean;
		toggleExpanded: (rowId: string) => void;
		showWildcard: () => boolean;
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
				<div class="flex items-center gap-2">
					<Link
						to="/jobs/$id"
						params={{ id: info.row.original.ID }}
						class="block max-w-[260px] truncate font-medium text-foreground transition-colors hover:text-primary"
						title={info.getValue() as string}
					>
						{info.getValue() as string}
					</Link>
					<Show
						when={
							info.row.original.Wildcard &&
							info.table.options.meta?.showWildcard()
						}
					>
						<Badge variant="secondary" class="shrink-0">
							Wildcard
						</Badge>
					</Show>
				</div>
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
			cell: (info) => (
				<Show
					when={info.getValue() as string | null | undefined}
					fallback={<span class="text-faint">—</span>}
				>
					{(val) => (
						<span class="block max-w-[160px] truncate text-muted" title={val()}>
							{val()}
						</span>
					)}
				</Show>
			),
		},
		{
			accessorKey: "DaysInOffice",
			header: "Office",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => (
				<Switch fallback={<span class="text-faint">—</span>}>
					<Match when={info.getValue() === 0}>
						<span class="text-muted">Remote</span>
					</Match>
					<Match when={info.getValue() as number | null | undefined}>
						{(val) => (
							<span class="font-mono text-xs tabular-nums text-muted">
								{val()}d/wk
							</span>
						)}
					</Match>
				</Switch>
			),
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
						class={`rounded-md transition-all hover:opacity-80${expanded() ? " ring-2 ring-primary ring-offset-1" : ""}`}
					>
						{val != null ? (
							<SuitabilityScoreValue
								score={val}
								band={info.row.original.Band}
							/>
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
			cell: (info) => (
				<Show
					when={ctx.appsForJobs()?.[info.row.original.ID]}
					fallback={<span class="text-faint">—</span>}
				>
					{(summary) => (
						<Show
							when={summary().StatusName}
							fallback={<span class="text-faint">—</span>}
						>
							<StatusBadge
								name={summary().StatusName}
								colour={summary().StatusColour}
							/>
						</Show>
					)}
				</Show>
			),
		},
		{
			id: "actions",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => (
				<div class="flex items-center justify-end">
					<button
						type="button"
						aria-label="Not for me"
						title="Not for me"
						onClick={() => ctx.onDismiss(info.row.original)}
						class="flex h-8 w-8 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-foreground"
					>
						<Icon name="x" />
					</button>
					<JobActionsMenu
						job={info.row.original}
						appSummary={ctx.appsForJobs()?.[info.row.original.ID]}
						onTrack={() => ctx.onTrack(info.row.original.ID)}
					/>
				</div>
			),
		},
	];
}
