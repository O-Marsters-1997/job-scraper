import { Index, type JSX, onMount, Show } from "solid-js";
import { cn } from "@/lib/utils";

export type PickItem = { id: string; body: JSX.Element; meta: string };

export type PickListProps = {
	label: string;
	items: PickItem[];
	picked: string[];
	selected: string | undefined;
	onSelect: (id: string | undefined) => void;
	onToggle: (id: string) => void;
	onMove: (id: string, place: number) => void;
};

export function PickList(props: PickListProps) {
	let list: HTMLDivElement | undefined;
	onMount(() => list?.focus({ preventScroll: true }));
	const rank = (id: string) => props.picked.indexOf(id) + 1;
	const select = (id: string | undefined) => {
		props.onSelect(id);
		list
			?.querySelector(`[data-id="${CSS.escape(id ?? "")}"]`)
			?.scrollIntoView({ block: "nearest" });
	};
	const step = (by: number) => {
		const ids = props.items.map((s) => s.id);
		const at = props.selected ? ids.indexOf(props.selected) : -1;
		select(
			at < 0
				? ids.at(by > 0 ? 0 : -1)
				: ids[Math.min(Math.max(at + by, 0), ids.length - 1)],
		);
	};
	const onSelectedKey = (e: KeyboardEvent, id: string) => {
		if (e.key === "Escape") {
			e.stopPropagation();
			select(undefined);
		} else if (/^[1-9]$/.test(e.key)) {
			e.preventDefault();
			props.onMove(id, Number(e.key));
		} else if (e.key === " " || e.key === "Enter") {
			e.preventDefault();
			props.onToggle(id);
		}
	};
	const onKeyDown = (e: KeyboardEvent) => {
		if (e.key === "ArrowDown" || e.key === "ArrowUp") {
			e.preventDefault();
			step(e.key === "ArrowDown" ? 1 : -1);
		} else if (props.selected) onSelectedKey(e, props.selected);
	};

	return (
		<div
			ref={list}
			role="listbox"
			tabIndex={0}
			aria-label={props.label}
			aria-activedescendant={
				props.selected ? `pick-row-${props.selected}` : undefined
			}
			class="scroll-slim flex-1 overflow-y-auto px-3 py-2 focus-visible:outline-none"
			onKeyDown={onKeyDown}
			onClick={(e) => {
				const id =
					e.target instanceof Element
						? e.target.closest<HTMLElement>("[data-id]")?.dataset.id
						: undefined;
				if (id) {
					if (rank(id) === 0) props.onToggle(id);
					select(id);
				}
				list?.focus({ preventScroll: true });
			}}
		>
			<Index each={props.items}>
				{(item) => (
					<div
						id={`pick-row-${item().id}`}
						data-id={item().id}
						role="option"
						tabIndex={-1}
						aria-selected={props.selected === item().id}
						class={cn(
							"relative flex cursor-pointer items-start gap-3 rounded-md px-2 py-1.5 text-sm transition-colors outline-none hover:bg-surface-muted",
							rank(item().id) > 0 ? "text-foreground" : "text-muted",
							props.selected === item().id && "bg-surface-muted",
						)}
					>
						<Show when={props.selected === item().id && rank(item().id) > 0}>
							<span
								aria-hidden="true"
								class="absolute inset-y-1 left-0 w-0.5 rounded-full bg-primary"
							/>
						</Show>
						<button
							type="button"
							tabIndex={-1}
							aria-label={rank(item().id) > 0 ? "Remove" : "Add"}
							class={cn(
								"mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border font-mono text-2xs tabular-nums transition-colors",
								rank(item().id) > 0
									? "border-primary bg-primary text-primary-foreground hover:bg-primary-hover"
									: "border-border-strong hover:border-primary",
							)}
							onClick={(e) => {
								e.stopPropagation();
								props.onToggle(item().id);
								list?.focus({ preventScroll: true });
							}}
						>
							{rank(item().id) || ""}
						</button>
						<span class="flex-1">{item().body}</span>
						<span class="shrink-0 text-right font-mono text-xs tabular-nums text-faint">
							{item().meta}
						</span>
					</div>
				)}
			</Index>
		</div>
	);
}
