import type { JSX } from "solid-js";

export function Panel(props: { children: JSX.Element }) {
	return (
		<div class="rounded-xl border border-border bg-surface p-5">
			{props.children}
		</div>
	);
}
