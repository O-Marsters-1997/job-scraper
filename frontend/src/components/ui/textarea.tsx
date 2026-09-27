import { type ComponentProps, splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export function Textarea(props: ComponentProps<"textarea">) {
	const [local, others] = splitProps(props, ["class"]);
	return <textarea class={cn("field resize-none", local.class)} {...others} />;
}
