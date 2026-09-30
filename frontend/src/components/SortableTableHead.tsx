import type { JSX } from "solid-js";
import { Show } from "solid-js";
import { TableHead } from "@/components/ui/table";

export type SortDir = "asc" | "desc" | "none";

const ariaSort = {
	asc: "ascending",
	desc: "descending",
	none: "none",
} as const;

export function SortableTableHead(props: {
	sorted: SortDir | false;
	onToggle?: (e: MouseEvent) => void;
	children: JSX.Element;
}) {
	return (
		<TableHead
			aria-sort={props.sorted === false ? undefined : ariaSort[props.sorted]}
		>
			<Show when={props.sorted !== false} fallback={props.children}>
				<button
					type="button"
					onClick={(e) => props.onToggle?.(e)}
					class="flex items-center gap-1 uppercase tracking-wide select-none hover:text-foreground"
				>
					{props.children}
					<span class="text-faint" aria-hidden="true">
						{props.sorted === "asc" ? "↑" : props.sorted === "desc" ? "↓" : "↕"}
					</span>
				</button>
			</Show>
		</TableHead>
	);
}
