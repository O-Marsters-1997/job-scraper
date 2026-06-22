import { Select as SelectPrimitive } from "@kobalte/core/select";
import type { ComponentProps, ValidComponent } from "solid-js";
import { splitProps } from "solid-js";
import { cn } from "@/lib/utils";

// Re-export the root+namespace so callers can use Select<T>, Select.Value, etc.
export const Select = SelectPrimitive;

export type SelectTriggerProps<T extends ValidComponent = "button"> =
	ComponentProps<typeof SelectPrimitive.Trigger<T>>;

export const SelectTrigger = <T extends ValidComponent = "button">(
	props: SelectTriggerProps<T>,
) => {
	const [local, rest] = splitProps(props as SelectTriggerProps, [
		"class",
		"children",
	]);
	return (
		<SelectPrimitive.Trigger
			class={cn(
				"flex h-9 w-full items-center justify-between rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground focus:border-primary focus:outline-none disabled:cursor-not-allowed disabled:opacity-50 data-[placeholder]:text-faint",
				local.class,
			)}
			{...rest}
		>
			{local.children}
			<SelectPrimitive.Icon class="ml-1 shrink-0 opacity-50">
				<svg
					aria-hidden="true"
					width="12"
					height="12"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<polyline points="6 9 12 15 18 9" />
				</svg>
			</SelectPrimitive.Icon>
		</SelectPrimitive.Trigger>
	);
};

export type SelectContentProps = {
	class?: string;
};

export const SelectContent = (props: SelectContentProps) => (
	<SelectPrimitive.Portal>
		<SelectPrimitive.Content
			class={cn(
				"z-50 min-w-[8rem] origin-(--kb-select-content-transform-origin) overflow-hidden rounded-lg border border-border bg-surface shadow-xl",
				"data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0 data-[closed]:zoom-out-95 data-[expanded]:zoom-in-95",
				props.class,
			)}
		>
			<SelectPrimitive.Listbox class="p-1" />
		</SelectPrimitive.Content>
	</SelectPrimitive.Portal>
);

export type SelectItemProps<T extends ValidComponent = "li"> =
	ComponentProps<typeof SelectPrimitive.Item<T>>;

export const SelectItem = <T extends ValidComponent = "li">(
	props: SelectItemProps<T>,
) => {
	const [local, rest] = splitProps(props as SelectItemProps, ["class"]);
	return (
		<SelectPrimitive.Item
			class={cn(
				"relative flex cursor-pointer items-center rounded-sm px-3 py-2 text-sm text-foreground outline-none select-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent-subtle data-[highlighted]:text-foreground",
				local.class,
			)}
			{...rest}
		/>
	);
};

export const SelectItemLabel = SelectPrimitive.ItemLabel;
