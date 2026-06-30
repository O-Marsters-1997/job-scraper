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
	DialogRoot as Dialog,
	DialogTrigger,
	DialogCloseButton as DialogClose,
};

export type DialogOverlayProps<T extends ValidComponent = "div"> =
	ComponentProps<typeof DialogOverlayPrimitive<T>>;

export const DialogOverlay = <T extends ValidComponent = "div">(
	props: DialogOverlayProps<T>,
) => {
	const [local, rest] = splitProps(props as DialogOverlayProps, ["class"]);
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

export type DialogContentProps<T extends ValidComponent = "div"> =
	ComponentProps<typeof DialogContentPrimitive<T>>;

export const DialogContent = <T extends ValidComponent = "div">(
	props: DialogContentProps<T>,
) => {
	const [local, rest] = splitProps(props as DialogContentProps, ["class"]);
	return (
		<DialogPortal>
			<DialogOverlay />
			<DialogContentPrimitive
				class={cn(
					"fixed top-[50%] left-[50%] z-50 w-full max-w-md translate-x-[-50%] translate-y-[-50%] rounded-2xl border border-border bg-surface p-6 shadow-2xl",
					"data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0 data-[closed]:zoom-out-95 data-[expanded]:zoom-in-95 data-[closed]:slide-out-to-top-[2%] data-[expanded]:slide-in-from-top-[2%]",
					local.class,
				)}
				{...rest}
			/>
		</DialogPortal>
	);
};

export function DialogHeader(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div class={cn("mb-4 flex flex-col gap-1", local.class)} {...others} />
	);
}

export type DialogTitleProps<T extends ValidComponent = "h2"> = ComponentProps<
	typeof DialogTitlePrimitive<T>
>;

export const DialogTitle = <T extends ValidComponent = "h2">(
	props: DialogTitleProps<T>,
) => {
	const [local, rest] = splitProps(props as DialogTitleProps, ["class"]);
	return (
		<DialogTitlePrimitive
			class={cn("text-base font-semibold text-foreground", local.class)}
			{...rest}
		/>
	);
};

export function DialogFooter(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div
			class={cn("mt-5 flex items-center justify-end gap-2", local.class)}
			{...others}
		/>
	);
}
