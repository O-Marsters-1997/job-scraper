import type { ComponentProps } from "solid-js";
import { splitProps } from "solid-js";
import { cn } from "@/lib/utils";

export type TableProps = ComponentProps<"table">;

export const Table = (props: TableProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<div class="relative w-full overflow-x-auto">
			<table
				class={cn("w-full caption-bottom text-sm", local.class)}
				{...rest}
			/>
		</div>
	);
};

export type TableHeaderProps = ComponentProps<"thead">;

export const TableHeader = (props: TableHeaderProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<thead
			class={cn(
				"bg-surface-muted [&_tr]:border-b [&_tr]:border-border",
				local.class,
			)}
			{...rest}
		/>
	);
};

export type TableBodyProps = ComponentProps<"tbody">;

export const TableBody = (props: TableBodyProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<tbody class={cn("[&_tr:last-child]:border-0", local.class)} {...rest} />
	);
};

export type TableRowProps = ComponentProps<"tr">;

export const TableRow = (props: TableRowProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<tr
			class={cn(
				"border-b border-border transition-colors hover:bg-surface-muted",
				local.class,
			)}
			{...rest}
		/>
	);
};

export type TableHeadProps = ComponentProps<"th">;

export const TableHead = (props: TableHeadProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<th
			class={cn(
				"h-11 px-4 text-left align-middle text-xs font-semibold whitespace-nowrap text-faint uppercase tracking-wide",
				local.class,
			)}
			{...rest}
		/>
	);
};

export type TableCellProps = ComponentProps<"td">;

export const TableCell = (props: TableCellProps) => {
	const [local, rest] = splitProps(props, ["class"]);
	return (
		<td
			class={cn(
				"px-4 py-4 align-middle text-data text-foreground whitespace-nowrap",
				local.class,
			)}
			{...rest}
		/>
	);
};
