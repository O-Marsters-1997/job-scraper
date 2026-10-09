import { Show } from "solid-js";
import { Kbd } from "@/components/Kbd";
import { Button } from "@/components/ui/button";

export function Pager(props: {
	total: number;
	page: number;
	pageSize: number;
	noun: string;
	onPage: (page: number) => void;
	keyHints?: boolean;
}) {
	const pageCount = () => Math.max(1, Math.ceil(props.total / props.pageSize));
	const from = () => (props.page - 1) * props.pageSize + 1;
	const to = () => Math.min(props.page * props.pageSize, props.total);
	return (
		<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
			<span class="mr-auto text-xs text-faint" aria-live="polite">
				<Show when={props.total > 0} fallback={`No ${props.noun}`}>
					<span class="font-mono tabular-nums">
						{from().toLocaleString()}–{to().toLocaleString()}
					</span>{" "}
					of{" "}
					<span class="font-mono tabular-nums">
						{props.total.toLocaleString()}
					</span>{" "}
					{props.noun}
				</Show>
			</span>
			<Show when={pageCount() > 1}>
				<span class="hidden text-xs text-faint sm:inline">
					Page {props.page} of {pageCount()}
				</span>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page <= 1}
					title={props.keyHints ? "Previous page ([)" : undefined}
					onClick={() => props.onPage(props.page - 1)}
				>
					Previous
					<Show when={props.keyHints}>
						<Kbd>[</Kbd>
					</Show>
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={props.page >= pageCount()}
					title={props.keyHints ? "Next page (])" : undefined}
					onClick={() => props.onPage(props.page + 1)}
				>
					Next
					<Show when={props.keyHints}>
						<Kbd>]</Kbd>
					</Show>
				</Button>
			</Show>
		</div>
	);
}
