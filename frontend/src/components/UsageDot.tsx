import { Show } from "solid-js";
import { cn } from "@/lib/utils";
import type { UsageLevel } from "@/types/quota";

export function UsageDot(props: { level: UsageLevel; class?: string }) {
	return (
		<Show when={props.level !== "ok"}>
			<span
				role="img"
				aria-label="Usage warning"
				class={cn(
					"size-2 shrink-0 rounded-full",
					props.level === "critical" ? "bg-destructive" : "bg-status-interview",
					props.class,
				)}
			/>
		</Show>
	);
}
