import type { JSX } from "solid-js";
import { Show } from "solid-js";

export function PageHeading(props: {
	title: string;
	subtitle?: string;
	children?: JSX.Element;
}) {
	return (
		<div class="mb-5 flex items-start justify-between gap-4">
			<div>
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					{props.title}
				</h1>
				<Show when={props.subtitle}>
					<p class="mt-0.5 text-xs text-faint">{props.subtitle}</p>
				</Show>
			</div>
			{props.children}
		</div>
	);
}
