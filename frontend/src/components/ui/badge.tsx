import { type ComponentProps, splitProps } from "solid-js"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "../../lib/utils"

const badgeVariants = cva(
	"inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold",
	{
		variants: {
			variant: {
				default:
					"bg-[var(--lagoon)] text-white",
				secondary:
					"border border-[var(--chip-line)] bg-[var(--chip-bg)] text-[var(--sea-ink-soft)]",
				outline:
					"border border-[var(--line)] text-[var(--sea-ink-soft)]",
			},
		},
		defaultVariants: {
			variant: "default",
		},
	},
)

interface BadgeProps extends ComponentProps<"span">, VariantProps<typeof badgeVariants> {}

function Badge(props: BadgeProps) {
	const [local, others] = splitProps(props, ["class", "variant"])
	return (
		<span
			class={cn(badgeVariants({ variant: local.variant }), local.class)}
			{...others}
		/>
	)
}

export { Badge, badgeVariants }
