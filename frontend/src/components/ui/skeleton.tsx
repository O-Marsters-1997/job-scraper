import { For } from "solid-js";
import { cn } from "@/lib/utils";

export function Skeleton(props: { class?: string }) {
	return (
		<div
			aria-hidden="true"
			class={cn("animate-pulse rounded bg-surface-muted", props.class)}
		/>
	);
}

// Placeholder list matching the bordered, row-divided cards used across the
// app (applications, statuses, searches) while their data loads. Mirrors the
// loaded shape so the layout doesn't jump when content arrives.
export function SkeletonList(props: { rows?: number; class?: string }) {
	const count = () => props.rows ?? 4;
	return (
		<div
			aria-hidden="true"
			class={cn(
				"divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface",
				props.class,
			)}
		>
			<For each={Array.from({ length: count() })}>
				{() => (
					<div class="flex items-center gap-3 px-4 py-3.5">
						<Skeleton class="h-3 w-3 shrink-0 rounded-full" />
						<Skeleton class="h-4 w-[38%]" />
						<Skeleton class="ml-auto h-4 w-14" />
					</div>
				)}
			</For>
		</div>
	);
}
