import type { ComponentProps } from "solid-js"
import { splitProps } from "solid-js"
import { cn } from "@/lib/utils"

export type TableProps = ComponentProps<"table">

export const Table = (props: TableProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return (
		<div class="relative w-full overflow-x-auto">
			<table class={cn("w-full caption-bottom text-sm", local.class)} {...rest} />
		</div>
	)
}

export type TableHeaderProps = ComponentProps<"thead">

export const TableHeader = (props: TableHeaderProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return <thead class={cn("[&_tr]:border-b [&_tr]:border-[var(--line)]", local.class)} {...rest} />
}

export type TableBodyProps = ComponentProps<"tbody">

export const TableBody = (props: TableBodyProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return <tbody class={cn("[&_tr:last-child]:border-0", local.class)} {...rest} />
}

export type TableRowProps = ComponentProps<"tr">

export const TableRow = (props: TableRowProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return (
		<tr
			class={cn(
				"border-b border-[var(--line)] transition-colors hover:bg-[rgba(79,184,178,0.05)]",
				local.class,
			)}
			{...rest}
		/>
	)
}

export type TableHeadProps = ComponentProps<"th">

export const TableHead = (props: TableHeadProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return (
		<th
			class={cn(
				"h-10 px-3 text-left align-middle text-xs font-semibold text-[var(--sea-ink-soft)] whitespace-nowrap uppercase tracking-wide",
				local.class,
			)}
			{...rest}
		/>
	)
}

export type TableCellProps = ComponentProps<"td">

export const TableCell = (props: TableCellProps) => {
	const [local, rest] = splitProps(props, ["class"])
	return (
		<td
			class={cn("px-3 py-3 align-middle text-sm text-[var(--sea-ink)]", local.class)}
			{...rest}
		/>
	)
}
