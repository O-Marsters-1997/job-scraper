import type { JSX } from "solid-js";
import { Show } from "solid-js";
import { Dynamic } from "solid-js/web";

export function PageHeading(props: {
	title: string;
	level?: "h1" | "h2";
	subtitle?: string;
	children?: JSX.Element;
}) {
	return (
		<div class="mb-5 flex items-start justify-between gap-4">
			<div>
				<Dynamic
					component={props.level ?? "h1"}
					class="text-lg font-bold tracking-tight text-foreground"
				>
					{props.title}
				</Dynamic>
				<Show when={props.subtitle}>
					<p class="mt-0.5 text-xs text-faint">{props.subtitle}</p>
				</Show>
			</div>
			{props.children}
		</div>
	);
}
