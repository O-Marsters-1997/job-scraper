import { Link } from "@tanstack/solid-router";
import { createColumnHelper } from "@tanstack/solid-table";
import { Match, Show, Switch } from "solid-js";
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
	interface TableMeta<TData> {
		isExpanded: (rowId: string) => boolean;
		toggleExpanded: (rowId: string) => void;
	}
}

const col = createColumnHelper<Job>();

export function createJobColumns(ctx: JobTableContext) {
	return [
		col.accessor("Title", {
			header: "Title",
			cell: (info) => (
				<Link
					to="/jobs/$id"
					params={{ id: info.row.original.ID }}
					class="block max-w-[260px] truncate font-medium text-foreground transition-colors hover:text-primary"
					title={info.getValue()}
				>
					{info.getValue()}
				</Link>
			),
		}),
		col.accessor("CompanySlug", {
			header: "Company",
			cell: (info) => (
				<span
					class="block max-w-[180px] truncate text-muted"
					title={titleCase(info.getValue())}
				>
					{titleCase(info.getValue())}
				</span>
			),
		}),
		col.accessor("Location", {
			header: "Location",
			cell: (info) => (
				<Show
					when={info.getValue()}
					fallback={<span class="text-faint">—</span>}
				>
					{(val) => (
						<span class="block max-w-[160px] truncate text-muted" title={val()}>
							{val()}
						</span>
					)}
				</Show>
			),
		}),
		col.accessor("DaysInOffice", {
			header: "Office",
			enableSorting: false,
			enableGlobalFilter: false,
			cell: (info) => (
				<Switch fallback={<span class="text-faint">—</span>}>
					<Match when={info.getValue() === 0}>
						<span class="text-muted">Remote</span>
					</Match>
					<Match when={info.getValue()}>
						{(val) => (
							<span class="font-mono text-xs tabular-nums text-muted">
								{val()}d/wk
							</span>
						)}
					</Match>
				</Switch>
			),
		}),
		col.accessor("SuitabilityScore", {
			header: "Suitability",
			enableGlobalFilter: false,
			sortUndefined: -1,
			cell: (info) => {
				const val = info.getValue();
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
		}),
		col.accessor("Source", {
			header: "Source",
			cell: (info) => <SourceBadge source={info.getValue()} />,
		}),
		col.accessor("ScrapedAt", {
			header: "Scraped",
			enableGlobalFilter: false,
			cell: (info) => (
				<span class="font-mono text-xs tabular-nums text-faint">
					{formatRelative(info.getValue())}
				</span>
			),
		}),
		col.display({
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
								colour={summary().StatusColour || STATUS_FALLBACK_COLOUR}
							/>
						</Show>
					)}
				</Show>
			),
		}),
		col.display({
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
		}),
	];
}
