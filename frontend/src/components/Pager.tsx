import { Show } from "solid-js";
import { Button } from "@/components/ui/button";

export function Pager(props: {
	from: number;
	to: number;
	total: number;
	page: number;
	pageCount: number;
	noun: string;
	onPage: (page: number) => void;
}) {
	return (
		<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
			<span class="mr-auto text-xs text-faint" aria-live="polite">
				<Show when={props.total > 0} fallback={`No ${props.noun}`}>
					<span class="font-mono tabular-nums">
						{props.from.toLocaleString()}–{props.to.toLocaleString()}
					</span>{" "}
					of{" "}
					<span class="font-mono tabular-nums">
						{props.total.toLocaleString()}
					</span>{" "}
					{props.noun}
				</Show>
			</span>
			<Show when={props.pageCount > 1}>
				<span class="hidden text-xs text-faint sm:inline">
					Page {props.page} of {props.pageCount}
				</span>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page <= 1}
					onClick={() => props.onPage(props.page - 1)}
				>
					Previous
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page >= props.pageCount}
					onClick={() => props.onPage(props.page + 1)}
				>
					Next
				</Button>
			</Show>
		</div>
	);
}
