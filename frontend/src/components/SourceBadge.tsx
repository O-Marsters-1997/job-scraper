import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "solid-js";
import { splitProps } from "solid-js";
import { badgeVariants } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

const SOURCE_CLASSES = {
	LinkedIn: "bg-source-linkedin/10 text-source-linkedin",
	Indeed: "bg-source-indeed/10 text-source-indeed",
	Greenhouse: "bg-source-greenhouse/10 text-source-greenhouse",
	Lever: "bg-source-lever/10 text-source-lever",
};

const sourceColorVariants = cva("", {
	variants: { source: SOURCE_CLASSES },
});

type SourceKey = NonNullable<
	VariantProps<typeof sourceColorVariants>["source"]
>;

function isSourceKey(value: string): value is SourceKey {
	return Object.hasOwn(SOURCE_CLASSES, value);
}

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
				isSourceKey(local.source) &&
					sourceColorVariants({ source: local.source }),
				local.class,
			)}
			{...others}
		>
			{local.source}
		</span>
	);
}
