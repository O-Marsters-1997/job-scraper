import type { JSX } from "solid-js";

export function FactRow(props: {
	label: string;
	value?: string;
	children?: JSX.Element;
}) {
	return (
		<div class="flex items-start justify-between gap-4">
			<span class="shrink-0 text-xs text-faint">{props.label}</span>
			<span class="text-right text-xs text-foreground">
				{props.children ?? props.value}
			</span>
		</div>
	);
}
