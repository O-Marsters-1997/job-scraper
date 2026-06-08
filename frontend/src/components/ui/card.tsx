import { type ComponentProps, splitProps } from "solid-js";
import { cn } from "@/lib/utils";

function Card(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div
			class={cn(
				"flex flex-col rounded-xl border border-border bg-surface",
				local.class,
			)}
			{...others}
		/>
	);
}

function CardHeader(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div class={cn("flex flex-col gap-2 p-5 pb-3", local.class)} {...others} />
	);
}

function CardTitle(props: ComponentProps<"h3">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<h3
			class={cn(
				"text-base font-semibold leading-snug text-foreground",
				local.class,
			)}
			{...others}
		/>
	);
}

function CardContent(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return (
		<div class={cn("px-5 pb-3 flex flex-col gap-1", local.class)} {...others} />
	);
}

function CardFooter(props: ComponentProps<"div">) {
	const [local, others] = splitProps(props, ["class"]);
	return <div class={cn("px-5 pb-5 pt-2 mt-auto", local.class)} {...others} />;
}

export { Card, CardHeader, CardTitle, CardContent, CardFooter };
