import { type JSX, onCleanup, onMount, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import type { RoleBudgetValue } from "@/lib/tailoring";
import { cn } from "@/lib/utils";

export function LineBudget(props: { used: number; budget: number }) {
	const over = () => props.used > props.budget;
	return (
		<span
			data-testid="line-budget"
			class={cn(
				"flex items-center gap-2 font-mono text-xs tabular-nums",
				over() ? "text-destructive-strong" : "text-muted",
			)}
		>
			<span class="h-1.5 w-28 overflow-hidden rounded-full bg-border">
				<span
					class={cn("block h-full", over() ? "bg-destructive" : "bg-primary")}
					style={{
						width: `${Math.min(100, (props.used / Math.max(props.budget, 1)) * 100)}%`,
					}}
				/>
			</span>
			{props.used} / {props.budget} lines
		</span>
	);
}

export function RoleBudget(props: { budget: RoleBudgetValue }) {
	return (
		<span
			data-testid="role-budget"
			class={cn(
				"inline-flex items-center gap-1.5 font-mono text-xs tabular-nums",
				props.budget.over > 0 ? "text-destructive-strong" : "text-muted",
			)}
		>
			<Show when={props.budget.over > 0}>
				<span aria-hidden="true" class="size-1.5 rounded-full bg-destructive" />
			</Show>
			{props.budget.over > 0
				? `${props.budget.over} over`
				: `${props.budget.used}/${props.budget.budget} lines`}
		</span>
	);
}

export function Drawer(props: {
	label: string;
	title: string;
	hint: string;
	onClose: () => void;
	children: JSX.Element;
}) {
	let drawer: HTMLElement | undefined;
	onMount(() => {
		const onKey = (e: KeyboardEvent) => e.key === "Escape" && props.onClose();
		const onOutside = (e: MouseEvent) => {
			if (!(e.target instanceof Node) || !drawer?.contains(e.target))
				props.onClose();
		};
		window.addEventListener("keydown", onKey);
		document.addEventListener("mousedown", onOutside);
		onCleanup(() => {
			window.removeEventListener("keydown", onKey);
			document.removeEventListener("mousedown", onOutside);
		});
	});
	return (
		<aside
			ref={drawer}
			aria-label={props.label}
			class="fixed inset-y-0 right-0 z-[60] flex w-[30rem] flex-col border-l border-border bg-surface shadow-2xl"
		>
			<header class="flex items-start gap-3 border-b border-border px-5 py-4">
				<div class="flex-1">
					<h2 class="text-sm font-semibold text-foreground">{props.title}</h2>
					<p class="mt-0.5 text-xs text-muted">{props.hint}</p>
				</div>
				<button
					type="button"
					aria-label="Close"
					class="grid size-8 place-items-center rounded-md text-muted hover:bg-accent-subtle hover:text-foreground"
					onClick={() => props.onClose()}
				>
					<Icon name="x" size={16} />
				</button>
			</header>
			{props.children}
		</aside>
	);
}
