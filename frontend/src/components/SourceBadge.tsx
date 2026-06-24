import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "solid-js";
import { splitProps } from "solid-js";
import { badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const sourceColorVariants = cva("", {
	variants: {
		source: {
			LinkedIn: "bg-source-linkedin/10 text-source-linkedin",
			Indeed: "bg-source-indeed/10 text-source-indeed",
			Greenhouse: "bg-source-greenhouse/10 text-source-greenhouse",
			Lever: "bg-source-lever/10 text-source-lever",
		},
	},
});

type SourceKey = NonNullable<
	VariantProps<typeof sourceColorVariants>["source"]
>;

const KNOWN_SOURCES = new Set<string>([
	"LinkedIn",
	"Indeed",
	"Greenhouse",
	"Lever",
]);

interface SourceBadgeProps extends ComponentProps<"span"> {
	source: string;
}

export function SourceBadge(props: SourceBadgeProps) {
	const [local, others] = splitProps(props, ["source", "class"]);
	return (
		<span
			class={cn(
				badgeVariants({ variant: "source" }),
				"px-1.5 py-px text-2xs tracking-wider",
				KNOWN_SOURCES.has(local.source) &&
					sourceColorVariants({ source: local.source as SourceKey }),
				local.class,
			)}
			{...others}
		>
			{local.source}
		</span>
	);
}
