import type { JSX } from "solid-js";
import { Show } from "solid-js";

export function PageLayout(props: {
	title: string;
	actions?: JSX.Element;
	children: JSX.Element;
}) {
	return (
		<>
			<header class="sticky top-0 z-20 flex min-h-14 items-center justify-between gap-4 border-b border-border bg-background px-7 py-2.5">
				<h1 class="truncate text-lg font-bold tracking-tight text-foreground">
					{props.title}
				</h1>
				<Show when={props.actions}>
					<div class="flex shrink-0 items-center gap-2">{props.actions}</div>
				</Show>
			</header>
			<div class="px-7 py-6">{props.children}</div>
		</>
	);
}
