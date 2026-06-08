import { type ComponentProps, splitProps } from "solid-js";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

const labelVariants = cva("mb-1 block text-xs font-medium text-muted", {
	variants: {
		variant: {
			default: "",
			uppercase: "font-semibold uppercase tracking-wide",
		},
	},
	defaultVariants: {
		variant: "default",
	},
});

export type LabelProps = ComponentProps<"label"> &
	VariantProps<typeof labelVariants>;

export function Label(props: LabelProps) {
	const [local, others] = splitProps(props, ["class", "variant"]);
	return (
		<label
			class={cn(labelVariants({ variant: local.variant }), local.class)}
			{...others}
		/>
	);
}
