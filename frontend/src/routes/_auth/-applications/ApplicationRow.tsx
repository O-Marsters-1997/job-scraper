import { Link } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { StatusBadge } from "@/components/StatusBadge";
import { TableCell, TableRow } from "@/components/ui/table";
import { formatChaseDate } from "@/lib/chase";
import { CV_STAGE_LABEL, cvStage } from "@/lib/cvStage";
import { cn } from "@/lib/utils";
import type { ApplicationWithDetails } from "@/types/application";

export function ApplicationRow(props: {
	app: ApplicationWithDetails;
	selected: boolean;
	highlightOverdue: boolean;
	overdue: boolean;
	statusColour: string;
	chasePending: boolean;
	onToggle: (extendRange: boolean) => void;
	onChased: () => void;
	onEdit: () => void;
	onDelete: () => void;
}) {
	const failed = () => cvStage(props.app.Draft) === "failed";
	return (
		<TableRow
			data-state={props.selected ? "selected" : undefined}
			class={cn(
				props.selected && "bg-accent-subtle/40",
				props.highlightOverdue && props.overdue && "bg-destructive-subtle",
			)}
		>
			<TableCell>
				<input
					type="checkbox"
					aria-label={`Select ${props.app.JobTitle}`}
					class="size-4 cursor-pointer accent-primary"
					checked={props.selected}
					onMouseDown={(e) => {
						if (e.shiftKey) e.preventDefault();
					}}
					onClick={(e) => {
						props.onToggle(e.shiftKey);
						e.currentTarget.checked = props.selected;
					}}
				/>
			</TableCell>
			<TableCell class="max-w-xs">
				<Link
					to="/applications/$id"
					params={{ id: props.app.ID }}
					title={props.app.JobTitle}
					class="block truncate font-medium text-foreground hover:underline"
				>
					{props.app.JobTitle}
				</Link>
				<p class="truncate text-xs text-faint">
					{props.app.JobCompanySlug}
					{props.app.JobLocation ? ` · ${props.app.JobLocation}` : ""}
				</p>
				<Show when={props.app.JobClosedAt}>
					<span class="mt-1 inline-block rounded-full border border-border px-2 py-0.5 text-xs text-muted">
						Listing closed
					</span>
				</Show>
			</TableCell>
			<TableCell>
				<Show when={props.app.StatusName}>
					<StatusBadge
						name={props.app.StatusName}
						colour={props.statusColour}
					/>
				</Show>
			</TableCell>
			<TableCell
				class={cn(
					"text-xs",
					failed() ? "text-destructive-strong" : "text-muted",
				)}
				title={props.app.Draft?.LastError || undefined}
			>
				{CV_STAGE_LABEL[cvStage(props.app.Draft)]}
			</TableCell>
			<TableCell class="font-mono text-xs tabular-nums text-faint">
				{props.app.AppliedAt}
			</TableCell>
			<TableCell
				class={cn(
					"font-mono text-xs tabular-nums",
					props.overdue ? "text-destructive-strong" : "text-faint",
				)}
			>
				<Show when={props.app.ChaseBy}>
					{(chaseBy) => formatChaseDate(chaseBy())}
				</Show>
			</TableCell>
			<TableCell class="text-right">
				<Show when={props.app.ChaseBy}>
					<button
						type="button"
						disabled={props.chasePending}
						onClick={() => props.onChased()}
						class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
					>
						Chased
					</button>
				</Show>
				<button
					type="button"
					aria-label={`Edit ${props.app.JobTitle}`}
					onClick={() => props.onEdit()}
					class="rounded px-2 py-1 text-xs font-medium text-primary transition hover:bg-accent-subtle"
				>
					Edit
				</button>
				<button
					type="button"
					aria-label={`Delete ${props.app.JobTitle}`}
					onClick={() => props.onDelete()}
					class="rounded px-2 py-1 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle"
				>
					Delete
				</button>
			</TableCell>
		</TableRow>
	);
}
