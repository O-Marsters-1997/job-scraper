import type { Resource } from "solid-js";
import { Show } from "solid-js";
import { cn } from "@/lib/utils";

export function PdfPreview(props: {
	url: Resource<string>;
	title: string;
	class?: string;
}) {
	const size = () => cn("h-[1120px] w-full", props.class);
	return (
		<>
			<Show when={props.url.loading}>
				<div class={cn(size(), "animate-pulse rounded-xl bg-surface-muted")} />
			</Show>

			<Show when={props.url.error}>
				{(err) => (
					<div
						class={cn(
							"rounded-xl border border-destructive/30 bg-destructive-subtle p-6",
							props.class,
						)}
					>
						<p class="mb-1 text-sm font-semibold text-destructive-strong">
							Failed to load PDF
						</p>
						<p class="text-sm text-muted">
							{err() instanceof Error ? err().message : "Failed to load PDF"}
						</p>
					</div>
				)}
			</Show>

			<Show when={!props.url.loading && !props.url.error && props.url()}>
				{(url) => (
					<iframe
						src={url()}
						title={props.title}
						class={cn(size(), "rounded-xl border border-border bg-surface")}
					/>
				)}
			</Show>
		</>
	);
}
