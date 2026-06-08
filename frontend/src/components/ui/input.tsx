import { type ComponentProps, splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export function Input(props: ComponentProps<"input">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<input
			class={cn(
				"w-full rounded-md border border-border bg-surface px-3 py-2 text-sm text-foreground placeholder:text-faint focus:border-primary focus:ring-2 focus:ring-primary/10 focus:outline-none disabled:cursor-not-allowed disabled:opacity-50",
				local.class,
			)}
			{...others}
		/>
	);
}
