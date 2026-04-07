import type { ComponentProps, ValidComponent } from "solid-js"
import { splitProps } from "solid-js"
import { Root as ButtonPrimitive } from "@kobalte/core/button"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/lib/utils"

const buttonVariants = cva(
	"inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all shrink-0 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--lagoon)] focus-visible:ring-offset-1",
	{
		variants: {
			variant: {
				default: "bg-[var(--lagoon)] text-white hover:bg-[var(--lagoon-deep)]",
				destructive: "bg-red-500 text-white hover:bg-red-500/90",
				outline:
					"border border-[var(--line)] bg-transparent hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]",
				secondary:
					"bg-[rgba(79,184,178,0.14)] text-[var(--lagoon-deep)] hover:bg-[rgba(79,184,178,0.24)]",
				ghost: "hover:bg-[rgba(79,184,178,0.1)] hover:text-[var(--sea-ink)]",
				link: "text-[var(--lagoon-deep)] underline-offset-4 hover:underline",
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
)

export type ButtonProps<T extends ValidComponent = "button"> = ComponentProps<
	typeof ButtonPrimitive<T>
> &
	VariantProps<typeof buttonVariants>

export const Button = <T extends ValidComponent = "button">(props: ButtonProps<T>) => {
	const [local, rest] = splitProps(props as ButtonProps, ["class", "variant", "size"])
	return (
		<ButtonPrimitive
			class={cn(buttonVariants({ variant: local.variant, size: local.size }), local.class)}
			{...rest}
		/>
	)
}
