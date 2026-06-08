import {
	type ColumnDef,
	createSolidTable,
	flexRender,
	getCoreRowModel,
	getFilteredRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type PaginationState,
	type SortingState,
} from "@tanstack/solid-table";
import { createSignal, For, Show } from "solid-js";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

interface JobsDataTableProps<TData> {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
}

// Returns a windowed list of page numbers with null for ellipsis gaps.
function pageWindow(current: number, total: number): (number | null)[] {
	if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
	const pages = new Set([
		1,
		Math.max(1, current - 1),
		current,
		Math.min(total, current + 1),
		total,
	]);
	const sorted = [...pages].sort((a, b) => a - b);
	const result: (number | null)[] = [];
	for (let i = 0; i < sorted.length; i++) {
		if (i > 0 && (sorted[i] as number) - (sorted[i - 1] as number) > 1) {
			result.push(null);
		}
		result.push(sorted[i] as number);
	}
	return result;
}

export function JobsDataTable<TData>(props: JobsDataTableProps<TData>) {
	const [globalFilter, setGlobalFilter] = createSignal("");
	const [sorting, setSorting] = createSignal<SortingState>([]);
	const [pagination, setPagination] = createSignal<PaginationState>({
		pageIndex: 0,
		pageSize: 10,
	});

	const table = createSolidTable({
		get data() {
			return props.data;
		},
		columns: props.columns,
		getCoreRowModel: getCoreRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		globalFilterFn: "includesString",
		state: {
			get globalFilter() {
				return globalFilter();
			},
			get sorting() {
				return sorting();
			},
			get pagination() {
				return pagination();
			},
		},
		onGlobalFilterChange: setGlobalFilter,
		onSortingChange: setSorting,
		onPaginationChange: setPagination,
	});

	const pageIndex = () => table.getState().pagination.pageIndex;
	const pageCount = () => table.getPageCount();
	const filteredCount = () => table.getFilteredRowModel().rows.length;
	const start = () => pageIndex() * pagination().pageSize + 1;
	const end = () =>
		Math.min((pageIndex() + 1) * pagination().pageSize, filteredCount());

	const pgBtnClass = (active: boolean, disabled: boolean) =>
		cn(
			"flex size-7 items-center justify-center rounded-md border text-xs font-medium transition-colors",
			active
				? "border-primary bg-primary text-primary-foreground"
				: "border-border text-muted hover:border-border-strong hover:text-foreground",
			disabled && "opacity-40",
		);

	return (
		<div class="flex flex-col gap-3">
			{/* Search */}
			<div class="relative max-w-xs">
				<svg
					aria-hidden="true"
					class="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-faint"
					width="14"
					height="14"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					<circle cx="11" cy="11" r="8" />
					<line x1="21" y1="21" x2="16.65" y2="16.65" />
				</svg>
				<Input
					type="search"
					placeholder="Search by role or company…"
					value={globalFilter()}
					onInput={(e) => {
						setGlobalFilter(e.currentTarget.value);
						setPagination((p) => ({ ...p, pageIndex: 0 }));
					}}
					class="pr-3 pl-9"
				/>
			</div>

			{/* Table card — pagination lives inside so it shares the rounded border */}
			<div class="overflow-hidden rounded-xl border border-border bg-surface">
				<Table>
					<TableHeader>
						<For each={table.getHeaderGroups()}>
							{(headerGroup) => (
								<TableRow class="hover:bg-transparent">
									<For each={headerGroup.headers}>
										{(header) => (
											<TableHead
												class={
													header.column.getCanSort()
														? "cursor-pointer select-none"
														: undefined
												}
												onClick={header.column.getToggleSortingHandler()}
											>
												<div class="flex items-center gap-1">
													{header.isPlaceholder
														? null
														: flexRender(
																header.column.columnDef.header,
																header.getContext(),
															)}
													<Show when={header.column.getCanSort()}>
														<span class="text-faint">
															{header.column.getIsSorted() === "asc"
																? "↑"
																: header.column.getIsSorted() === "desc"
																	? "↓"
																	: "↕"}
														</span>
													</Show>
												</div>
											</TableHead>
										)}
									</For>
								</TableRow>
							)}
						</For>
					</TableHeader>
					<TableBody>
						<Show
							when={table.getRowModel().rows.length > 0}
							fallback={
								<tr>
									<td
										colspan={props.columns.length}
										class="py-12 text-center text-sm text-muted"
									>
										No jobs found.
									</td>
								</tr>
							}
						>
							<For each={table.getRowModel().rows}>
								{(row) => (
									<TableRow>
										<For each={row.getVisibleCells()}>
											{(cell) => (
												<TableCell>
													{flexRender(
														cell.column.columnDef.cell,
														cell.getContext(),
													)}
												</TableCell>
											)}
										</For>
									</TableRow>
								)}
							</For>
						</Show>
					</TableBody>
				</Table>

				{/* Numbered pagination — inside card, separated by a top border */}
				<Show when={pageCount() > 1}>
					<div class="flex items-center justify-between border-t border-border px-4 py-2.5">
						<p class="text-xs text-faint">
							Showing {start()}–{end()} of {filteredCount()} jobs
						</p>
						<div class="flex items-center gap-1">
							<button
								type="button"
								onClick={() => table.previousPage()}
								disabled={!table.getCanPreviousPage()}
								class={pgBtnClass(false, !table.getCanPreviousPage())}
								aria-label="Previous page"
							>
								←
							</button>

							<For each={pageWindow(pageIndex() + 1, pageCount())}>
								{(p) =>
									p === null ? (
										<span class="flex size-7 items-center justify-center text-xs text-faint">
											…
										</span>
									) : (
										<button
											type="button"
											onClick={() => table.setPageIndex((p as number) - 1)}
											class={pgBtnClass(p === pageIndex() + 1, false)}
											aria-label={`Page ${p}`}
											aria-current={p === pageIndex() + 1 ? "page" : undefined}
										>
											{p}
										</button>
									)
								}
							</For>

							<button
								type="button"
								onClick={() => table.nextPage()}
								disabled={!table.getCanNextPage()}
								class={pgBtnClass(false, !table.getCanNextPage())}
								aria-label="Next page"
							>
								→
							</button>
						</div>
					</div>
				</Show>
			</div>
		</div>
	);
}
