import type { JSX } from "solid-js";
import { Show } from "solid-js";
import { TableHead } from "@/components/ui/table";

type SortState = "ascending" | "descending" | "none";

export function SortableTableHead(props: {
	sorted: SortState | false;
	onToggle?: JSX.EventHandlerUnion<HTMLButtonElement, MouseEvent>;
	children: JSX.Element;
}) {
	return (
		<TableHead aria-sort={props.sorted === false ? undefined : props.sorted}>
			<Show when={props.sorted !== false} fallback={props.children}>
				<button
					type="button"
					onClick={props.onToggle}
					class="flex items-center gap-1 select-none hover:text-foreground"
				>
					{props.children}
					<span class="text-faint" aria-hidden="true">
						{props.sorted === "ascending"
							? "↑"
							: props.sorted === "descending"
								? "↓"
								: "↕"}
					</span>
				</button>
			</Show>
		</TableHead>
	);
}
