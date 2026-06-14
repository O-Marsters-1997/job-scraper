import {
	Control as SwitchControlPrimitive,
	Label as SwitchLabelPrimitive,
	Root as SwitchRoot,
	Thumb as SwitchThumbPrimitive,
} from "@kobalte/core/switch";
import type { ComponentProps, ValidComponent } from "solid-js";
import { splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export { SwitchRoot as Switch };

export type SwitchLabelProps<T extends ValidComponent = "label"> =
	ComponentProps<typeof SwitchLabelPrimitive<T>>;

export const SwitchLabel = <T extends ValidComponent = "label">(
	props: SwitchLabelProps<T>,
) => {
	const [local, rest] = splitProps(props as SwitchLabelProps, ["class"]);
	return (
		<SwitchLabelPrimitive
			class={cn("cursor-pointer select-none text-sm text-muted", local.class)}
			{...rest}
		/>
	);
};

export type SwitchControlProps<T extends ValidComponent = "div"> =
	ComponentProps<typeof SwitchControlPrimitive<T>>;

export const SwitchControl = <T extends ValidComponent = "div">(
	props: SwitchControlProps<T>,
) => {
	const [local, rest] = splitProps(props as SwitchControlProps, ["class"]);
	return (
		<SwitchControlPrimitive
			class={cn(
				"inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent bg-border-strong transition-colors",
				"data-[checked]:bg-primary",
				"data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50",
				local.class,
			)}
			{...rest}
		/>
	);
};

export type SwitchThumbProps<T extends ValidComponent = "span"> =
	ComponentProps<typeof SwitchThumbPrimitive<T>>;

export const SwitchThumb = <T extends ValidComponent = "span">(
	props: SwitchThumbProps<T>,
) => {
	const [local, rest] = splitProps(props as SwitchThumbProps, ["class"]);
	return (
		<SwitchThumbPrimitive
			class={cn(
				"pointer-events-none block size-4 translate-x-0 rounded-full bg-surface shadow-sm transition-transform",
				"data-[checked]:translate-x-4",
				local.class,
			)}
			{...rest}
		/>
	);
};
