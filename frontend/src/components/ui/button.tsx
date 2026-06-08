import { Root as ButtonPrimitive } from "@kobalte/core/button";
import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps, ValidComponent } from "solid-js";
import { splitProps } from "solid-js";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
	"inline-flex shrink-0 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
	{
		variants: {
			variant: {
				default: "bg-primary text-primary-foreground hover:bg-primary-hover",
				destructive: "bg-destructive text-white hover:bg-destructive-strong",
				outline:
					"border border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
				secondary:
					"border border-accent-border bg-accent-subtle text-accent-text hover:bg-accent-subtle/70",
				ghost: "text-muted hover:bg-accent-subtle hover:text-foreground",
				link: "text-primary underline-offset-4 hover:underline",
			},
			size: {
				default: "h-9 px-4 py-2",
				sm: "h-8 rounded-md px-3",
				lg: "h-10 rounded-md px-6",
				icon: "h-9 w-9",
				"icon-sm": "h-8 w-8",
				"icon-lg": "h-10 w-10",
			},
		},
		defaultVariants: {
			variant: "default",
			size: "default",
		},
	},
);

export type ButtonProps<T extends ValidComponent = "button"> = ComponentProps<
	typeof ButtonPrimitive<T>
> &
	VariantProps<typeof buttonVariants>;

export const Button = <T extends ValidComponent = "button">(
	props: ButtonProps<T>,
) => {
	const [local, rest] = splitProps(props as ButtonProps, [
		"class",
		"variant",
		"size",
	]);
	return (
		<ButtonPrimitive
			class={cn(
				buttonVariants({ variant: local.variant, size: local.size }),
				local.class,
			)}
			{...rest}
		/>
	);
};
