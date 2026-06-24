import { type ComponentProps, splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export function Input(props: ComponentProps<"input">) {
	const [local, others] = splitProps(props, ["class"]);
	return <input class={cn("field", local.class)} {...others} />;
}
