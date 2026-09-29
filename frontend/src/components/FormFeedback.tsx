import { Show } from "solid-js";

export function FormFeedback(props: {
	success?: boolean | string;
	error?: string | null;
}) {
	return (
		<>
			<Show when={props.success}>
				<div class="mb-4 rounded-lg border border-primary/30 bg-accent-subtle px-4 py-3 text-sm text-primary">
					{typeof props.success === "string" ? props.success : "Saved."}
				</div>
			</Show>
			<Show when={props.error}>
				<div
					role="alert"
					class="mb-4 rounded-lg border border-destructive/30 bg-destructive-subtle px-4 py-3 text-sm text-destructive-strong"
				>
					{props.error}
				</div>
			</Show>
		</>
	);
}
