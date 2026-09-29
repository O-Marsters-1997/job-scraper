import { createEffect, createSignal, For, on, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

export function SearchField(props: {
	id: string;
	label: string;
	placeholder: string;
	value: string;
	onInput: (value: string) => void;
}) {
	return (
		<div class="relative w-full sm:max-w-64">
			<Icon
				name="search"
				size={14}
				class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
			/>
			<Label for={props.id} class="sr-only">
				{props.label}
			</Label>
			<Input
				id={props.id}
				type="search"
				placeholder={props.placeholder}
				value={props.value}
				onInput={(e) => props.onInput(e.currentTarget.value)}
				class="h-8 pr-3 pl-9 text-sm"
			/>
		</div>
	);
}

export function FilterChips<T extends string>(props: {
	label: string;
	options: { value: T; label: string; count?: number }[];
	value: T;
	onChange: (value: T) => void;
}) {
	return (
		<fieldset class="flex flex-wrap items-center gap-1">
			<legend class="sr-only">{props.label}</legend>
			<For each={props.options}>
				{(o) => (
					<button
						type="button"
						aria-pressed={props.value === o.value}
						onClick={() => props.onChange(o.value)}
						class={cn(
							"inline-flex h-7 items-center gap-1.5 rounded-full border px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
							props.value === o.value
								? "border-accent-border bg-accent-subtle text-accent-text"
								: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
						)}
					>
						{o.label}
						<Show when={o.count !== undefined}>
							<span class="font-mono tabular-nums opacity-70">{o.count}</span>
						</Show>
					</button>
				)}
			</For>
		</fieldset>
	);
}

export type SortDir = "asc" | "desc";
export interface Sort<K extends string> {
	key: K;
	dir: SortDir;
}

export const nextSort = <K extends string>(
	current: Sort<K>,
	key: K,
	firstDir: SortDir = "asc",
): Sort<K> =>
	current.key === key
		? { key, dir: current.dir === "asc" ? "desc" : "asc" }
		: { key, dir: firstDir };

export const ariaSort = <K extends string>(current: Sort<K>, key: K) =>
	current.key === key
		? current.dir === "asc"
			? ("ascending" as const)
			: ("descending" as const)
		: ("none" as const);

export function usePaged<T>(
	items: () => T[],
	pageSize: number,
	resetOn: () => unknown,
) {
	const [page, setPage] = createSignal(1);
	const pageCount = () => Math.max(1, Math.ceil(items().length / pageSize));
	const current = () => Math.min(page(), pageCount());
	createEffect(on(resetOn, () => setPage(1), { defer: true }));
	return {
		page: current,
		pageCount,
		total: () => items().length,
		pageItems: () =>
			items().slice((current() - 1) * pageSize, current() * pageSize),
		from: () => (items().length ? (current() - 1) * pageSize + 1 : 0),
		to: () => Math.min(current() * pageSize, items().length),
		prev: () => setPage(current() - 1),
		next: () => setPage(current() + 1),
	};
}

export type Paged = ReturnType<typeof usePaged>;

export function Pager(props: { paged: Paged; noun: string }) {
	return (
		<div class="flex items-center justify-end gap-2 border-t border-border px-4 py-2.5">
			<span class="mr-auto text-xs text-faint" aria-live="polite">
				<Show when={props.paged.total() > 0} fallback={`No ${props.noun}`}>
					<span class="font-mono tabular-nums">
						{props.paged.from()}–{props.paged.to()}
					</span>{" "}
					of <span class="font-mono tabular-nums">{props.paged.total()}</span>{" "}
					{props.noun}
				</Show>
			</span>
			<Show when={props.paged.pageCount() > 1}>
				<span class="hidden text-xs text-faint sm:inline">
					Page {props.paged.page()} of {props.paged.pageCount()}
				</span>
				<Button
					variant="outline"
					size="sm"
					disabled={props.paged.page() <= 1}
					onClick={props.paged.prev}
				>
					Previous
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={props.paged.page() >= props.paged.pageCount()}
					onClick={props.paged.next}
				>
					Next
				</Button>
			</Show>
		</div>
	);
}
