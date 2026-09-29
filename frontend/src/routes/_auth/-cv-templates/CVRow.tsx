import { Link, useNavigate } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Badge } from "@/components/ui/badge";
import { TableCell, TableRow } from "@/components/ui/table";
import { formatDate } from "@/lib/datetime";
import type { CV } from "@/types/cv";

export function CVRow(props: {
	cv: CV;
	hidePending: boolean;
	showPending: boolean;
	onHide: () => void;
	onRestore: () => void;
}) {
	const navigate = useNavigate();
	return (
		<TableRow
			class={`cursor-pointer${!props.cv.Visible ? " opacity-60" : ""}`}
			onClick={() => {
				navigate({
					to: "/cv-templates/$docId/$tabId",
					params: { docId: props.cv.DocID, tabId: props.cv.TabID },
				});
			}}
		>
			<TableCell class="py-2.5">
				<div class="flex items-center gap-2">
					<Link
						to="/cv-templates/$docId/$tabId"
						params={{ docId: props.cv.DocID, tabId: props.cv.TabID }}
						onClick={(e) => e.stopPropagation()}
						class="font-medium text-foreground underline-offset-2 hover:text-primary hover:underline"
					>
						{props.cv.Title || "—"}
					</Link>
					<a
						href={props.cv.DocURL}
						target="_blank"
						rel="noreferrer"
						onClick={(e) => e.stopPropagation()}
						title="Open in Google Docs"
						class="text-faint transition-colors hover:text-foreground"
					>
						<Icon name="externalLink" size={12} />
						<span class="sr-only">
							Open {props.cv.Title || "document"} in Google Docs
						</span>
					</a>
					<Show when={!props.cv.Visible}>
						<Badge variant="secondary">Hidden</Badge>
					</Show>
				</div>
			</TableCell>
			<TableCell class="py-2.5 text-muted">
				{props.cv.SourceDoc || "—"}
			</TableCell>
			<TableCell class="py-2.5">
				<span class="font-mono text-xs tabular-nums text-faint">
					{formatDate(props.cv.ModifiedAt)}
				</span>
			</TableCell>
			<TableCell
				class="py-2.5 text-right"
				onClick={(e) => e.stopPropagation()}
				onKeyDown={(e) => e.stopPropagation()}
			>
				<Show
					when={props.cv.Visible}
					fallback={
						<button
							type="button"
							title="Restore tab"
							onClick={() => props.onRestore()}
							disabled={props.showPending}
							class="inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-primary disabled:opacity-50"
						>
							<Icon name="rotateCcw" size={14} />
						</button>
					}
				>
					<button
						type="button"
						title="Hide tab"
						onClick={() => props.onHide()}
						disabled={props.hidePending}
						class="inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50"
					>
						<Icon name="trash" size={14} />
					</button>
				</Show>
			</TableCell>
		</TableRow>
	);
}
