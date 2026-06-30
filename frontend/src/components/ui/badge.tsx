import { cva, type VariantProps } from "class-variance-authority";
import { type ComponentProps, splitProps } from "solid-js";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
	"inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap",
	{
		variants: {
			variant: {
				default: "bg-primary text-primary-foreground",
				secondary: "bg-surface-muted text-muted",
				outline: "border border-border text-muted",
				source:
					"bg-surface-muted font-mono text-xs font-medium uppercase tracking-wide text-muted",
			},
		},
		defaultVariants: {
			variant: "default",
		},
	},
);

interface BadgeProps
	extends ComponentProps<"span">,
		VariantProps<typeof badgeVariants> {}

function Badge(props: BadgeProps) {
	const [local, others] = splitProps(props, ["class", "variant"]);
	return (
		<span
			class={cn(badgeVariants({ variant: local.variant }), local.class)}
			{...others}
		/>
	);
}

export { Badge, badgeVariants };
