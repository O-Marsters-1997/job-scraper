import type { JSX } from "solid-js";
import { cn } from "@/lib/utils";

export function Kbd(props: { class?: string; children: JSX.Element }) {
	return (
		<kbd
			class={cn(
				"inline-flex h-4 min-w-4 items-center justify-center rounded border border-border bg-surface-muted px-1 font-mono text-2xs text-faint",
				props.class,
			)}
		>
			{props.children}
		</kbd>
	);
}
