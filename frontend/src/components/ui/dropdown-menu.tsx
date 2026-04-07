import type { ComponentProps, ValidComponent } from "solid-js"
import { mergeProps, splitProps } from "solid-js"
import { DropdownMenu as DropdownMenuPrimitive } from "@kobalte/core/dropdown-menu"
import { cn } from "@/lib/utils"

export const DropdownMenuPortal = DropdownMenuPrimitive.Portal

export type DropdownMenuProps = ComponentProps<typeof DropdownMenuPrimitive>

export const DropdownMenu = (props: DropdownMenuProps) => {
	const merged = mergeProps<DropdownMenuProps[]>({ gutter: 4 }, props)
	return <DropdownMenuPrimitive {...merged} />
}

export type DropdownMenuTriggerProps<T extends ValidComponent = "button"> = ComponentProps<
	typeof DropdownMenuPrimitive.Trigger<T>
>

export const DropdownMenuTrigger = <T extends ValidComponent = "button">(
	props: DropdownMenuTriggerProps<T>,
) => {
	return <DropdownMenuPrimitive.Trigger {...props} />
}

export type DropdownMenuGroupProps<T extends ValidComponent = "div"> = ComponentProps<
	typeof DropdownMenuPrimitive.Group<T>
>

export const DropdownMenuGroup = <T extends ValidComponent = "div">(
	props: DropdownMenuGroupProps<T>,
) => {
	return <DropdownMenuPrimitive.Group {...props} />
}

export type DropdownMenuContentProps<T extends ValidComponent = "div"> = ComponentProps<
	typeof DropdownMenuPrimitive.Content<T>
>

export const DropdownMenuContent = <T extends ValidComponent = "div">(
	props: DropdownMenuContentProps<T>,
) => {
	const [local, rest] = splitProps(props as DropdownMenuContentProps, ["class"])
	return (
		<DropdownMenuPrimitive.Portal>
			<DropdownMenuPrimitive.Content
				class={cn(
					"z-50 min-w-[10rem] origin-(--kb-menu-content-transform-origin) overflow-hidden rounded-xl border border-[var(--line)] bg-white/95 py-1 shadow-lg backdrop-blur-sm",
					"data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0 data-[closed]:zoom-out-95 data-[expanded]:zoom-in-95",
					local.class,
				)}
				{...rest}
			/>
		</DropdownMenuPrimitive.Portal>
	)
}

export type DropdownMenuItemProps<T extends ValidComponent = "div"> = ComponentProps<
	typeof DropdownMenuPrimitive.Item<T>
> & {
	inset?: boolean
}

export const DropdownMenuItem = <T extends ValidComponent = "div">(
	props: DropdownMenuItemProps<T>,
) => {
	const [local, rest] = splitProps(props as DropdownMenuItemProps, ["class", "inset"])
	return (
		<DropdownMenuPrimitive.Item
			class={cn(
				"relative flex cursor-default select-none items-center gap-2 rounded-sm px-3 py-2 text-sm text-[var(--sea-ink)] outline-none transition-colors data-[highlighted]:bg-[rgba(79,184,178,0.1)] data-[highlighted]:text-[var(--sea-ink)] data-[disabled]:pointer-events-none data-[disabled]:opacity-50",
				local.inset && "pl-8",
				local.class,
			)}
			{...rest}
		/>
	)
}

export type DropdownMenuSeparatorProps<T extends ValidComponent = "hr"> = ComponentProps<
	typeof DropdownMenuPrimitive.Separator<T>
>

export const DropdownMenuSeparator = <T extends ValidComponent = "hr">(
	props: DropdownMenuSeparatorProps<T>,
) => {
	const [local, rest] = splitProps(props as DropdownMenuSeparatorProps, ["class"])
	return (
		<DropdownMenuPrimitive.Separator
			class={cn("-mx-1 my-1 h-px bg-[var(--line)]", local.class)}
			{...rest}
		/>
	)
}

export type DropdownMenuGroupLabelProps<T extends ValidComponent = "span"> = ComponentProps<
	typeof DropdownMenuPrimitive.GroupLabel<T>
>

export const DropdownMenuGroupLabel = <T extends ValidComponent = "span">(
	props: DropdownMenuGroupLabelProps<T>,
) => {
	const [local, rest] = splitProps(props as DropdownMenuGroupLabelProps, ["class"])
	return (
		<DropdownMenuPrimitive.GroupLabel
			as="div"
			class={cn("px-3 py-1.5 text-xs font-semibold text-[var(--sea-ink-soft)] uppercase tracking-wide", local.class)}
			{...rest}
		/>
	)
}
