import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import type { pipelineSegments } from "@/lib/overview";

export function PipelineCard(props: {
	segments: ReturnType<typeof pipelineSegments>;
}) {
	return (
		<div class="mb-4">
			<div class="mb-2 flex items-center justify-between">
				<span class="text-sm font-semibold text-foreground">
					Application pipeline
				</span>
				<Link
					to="/applications"
					search={{ status: undefined }}
					class="text-xs font-medium text-primary transition-colors hover:text-primary-hover"
				>
					View all →
				</Link>
			</div>
			<Show
				when={props.segments.length > 0}
				fallback={<p class="text-sm text-faint">No statuses configured.</p>}
			>
				<div class="flex gap-2">
					<For each={props.segments}>
						{(seg) => (
							<Link
								to="/applications"
								search={{ status: seg.ID }}
								class="flex min-w-0 flex-1 items-center gap-2.5 rounded-xl border border-border bg-surface px-4 py-3 transition-colors hover:border-border-strong hover:bg-surface-muted"
							>
								<span
									class="size-2 shrink-0 rounded-full"
									style={{ "background-color": seg.Colour }}
								/>
								<span class="min-w-0 flex-1 truncate text-xs font-medium text-muted">
									{seg.Name}
								</span>
								<span class="font-mono text-sm font-medium tabular-nums text-foreground">
									{seg.count}
								</span>
							</Link>
						)}
					</For>
				</div>
			</Show>
		</div>
	);
}
