import type { JSX } from "solid-js";
import { cn } from "@/lib/utils";

export function ToggleChip(props: {
	active: boolean;
	onClick: () => void;
	solid?: boolean;
	class?: string;
	children: JSX.Element;
}) {
	return (
		<button
			type="button"
			aria-pressed={props.active}
			onClick={() => props.onClick()}
			class={cn(
				"inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
				props.active
					? props.solid
						? "border-primary bg-primary text-primary-foreground"
						: "border-accent-border bg-accent-subtle text-accent-text"
					: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
				props.class,
			)}
		>
			{props.children}
		</button>
	);
}
