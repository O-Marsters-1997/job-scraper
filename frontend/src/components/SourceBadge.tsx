import type { ComponentProps } from "solid-js";
import { splitProps } from "solid-js";
import { badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

// Per-source colour overrides — appended after the base `source` variant so
// tailwind-merge replaces bg-surface-muted + text-muted with the correct tint.
const SOURCE_STYLES: Record<string, string> = {
	LinkedIn: "bg-source-linkedin/10 text-source-linkedin",
	Indeed: "bg-source-indeed/10 text-source-indeed",
	Greenhouse: "bg-source-greenhouse/10 text-source-greenhouse",
	Lever: "bg-source-lever/10 text-source-lever",
};

interface SourceBadgeProps extends ComponentProps<"span"> {
	source: string;
}

export function SourceBadge(props: SourceBadgeProps) {
	const [local, others] = splitProps(props, ["source", "class"]);
	const colorClass = () => SOURCE_STYLES[local.source] ?? "";
	return (
		<span
			class={cn(
				badgeVariants({ variant: "source" }),
				"px-1.5 py-px text-[10px] tracking-wider",
				colorClass(),
				local.class,
			)}
			{...others}
		>
			{local.source}
		</span>
	);
}
