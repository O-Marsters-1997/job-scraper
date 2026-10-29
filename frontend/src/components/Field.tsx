import { type JSX, Show } from "solid-js";

export function Field(props: {
	label: string;
	for: string;
	hint?: string;
	children: JSX.Element;
}) {
	return (
		<div>
			<label for={props.for} class="block text-sm font-medium text-foreground">
				{props.label}
			</label>
			<Show when={props.hint}>
				<p class="mt-0.5 text-xs text-faint">{props.hint}</p>
			</Show>
			<div class="mt-2">{props.children}</div>
		</div>
	);
}
