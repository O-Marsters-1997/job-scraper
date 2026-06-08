import {
	CloseButton as DialogCloseButton,
	Content as DialogContentPrimitive,
	Overlay as DialogOverlayPrimitive,
	Portal as DialogPortal,
	Root as DialogRoot,
	Title as DialogTitlePrimitive,
	Trigger as DialogTrigger,
} from "@kobalte/core/dialog";
import type { ComponentProps, ValidComponent } from "solid-js";
import { splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export {
	DialogRoot as Sheet,
	DialogTrigger as SheetTrigger,
	DialogCloseButton as SheetClose,
};

type SheetOverlayProps<T extends ValidComponent = "div"> = ComponentProps<
	typeof DialogOverlayPrimitive<T>
>;

const SheetOverlay = <T extends ValidComponent = "div">(
	props: SheetOverlayProps<T>,
) => {
	const [local, rest] = splitProps(props as SheetOverlayProps, ["class"]);
	return (
		<DialogOverlayPrimitive
			class={cn(
				"fixed inset-0 z-50 bg-black/30 backdrop-blur-[2px]",
				"data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0",
				local.class,
			)}
			{...rest}
		/>
	);
};

export type SheetContentProps<T extends ValidComponent = "div"> =
	ComponentProps<typeof DialogContentPrimitive<T>>;

export const SheetContent = <T extends ValidComponent = "div">(
	props: SheetContentProps<T>,
) => {
	const [local, rest] = splitProps(props as SheetContentProps, ["class"]);
	return (
		<DialogPortal>
			<SheetOverlay />
			<DialogContentPrimitive
				class={cn(
					"fixed inset-y-0 right-0 z-50 flex h-full w-80 flex-col border-l border-border bg-surface shadow-2xl",
					"data-[expanded]:animate-in data-[closed]:animate-out",
					"data-[expanded]:slide-in-from-right-full data-[closed]:slide-out-to-right-full",
					"data-[expanded]:fade-in-0 data-[closed]:fade-out-0",
					"data-[expanded]:duration-300 data-[closed]:duration-200",
					local.class,
				)}
				{...rest}
			/>
		</DialogPortal>
	);
};

export function SheetHeader(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div
			class={cn(
				"flex items-center justify-between border-b border-border px-6 py-4",
				local.class,
			)}
			{...others}
		/>
	);
}

export type SheetTitleProps<T extends ValidComponent = "h2"> = ComponentProps<
	typeof DialogTitlePrimitive<T>
>;

export const SheetTitle = <T extends ValidComponent = "h2">(
	props: SheetTitleProps<T>,
) => {
	const [local, rest] = splitProps(props as SheetTitleProps, ["class"]);
	return (
		<DialogTitlePrimitive
			class={cn("text-sm font-semibold text-foreground", local.class)}
			{...rest}
		/>
	);
};
