import {
	type ColumnDef,
	createSolidTable,
	flexRender,
	getCoreRowModel,
	getPaginationRowModel,
	getSortedRowModel,
	type PaginationState,
	type SortingState,
} from "@tanstack/solid-table";
import { createEffect, createSignal, For, Show } from "solid-js";
import { JobFiltersDialog } from "@/components/jobs/JobFiltersDialog";
import { JobRowExpander } from "@/components/jobs/JobRowExpander";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { activeFilterCount, type JobFilters } from "@/lib/jobFilters";
import { cn } from "@/lib/utils";
import type { Job } from "@/types/job";

interface JobsDataTableProps<TData extends Job> {
	columns: ColumnDef<TData, unknown>[];
	data: TData[];
	filters: JobFilters;
	onChange: (patch: Partial<JobFilters>) => void;
	sourceOptions: string[];
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

export function JobsDataTable<TData extends Job>(
	props: JobsDataTableProps<TData>,
) {
	const [sorting, setSorting] = createSignal<SortingState>([]);
	const [suitabilityMin, setSuitabilityMin] = createSignal("");
	const [showSkipped, setShowSkipped] = createSignal(true);
	const [pagination, setPagination] = createSignal<PaginationState>({
		pageIndex: 0,
		pageSize: 10,
	});
	const [filtersOpen, setFiltersOpen] = createSignal(false);

	// Reset to page 1 whenever filtered data changes length.
	createEffect(() => {
		void props.data.length;
		setPagination((p) => ({ ...p, pageIndex: 0 }));
	});

	// Per-row expand state — keyed by Job ID, isolated from sort/filter/pagination
	const [expandedRows, setExpandedRows] = createSignal<Set<string>>(new Set());
	const isExpanded = (rowId: string) => expandedRows().has(rowId);
	const toggleExpanded = (rowId: string) => {
		setExpandedRows((prev) => {
			const next = new Set(prev);
			if (next.has(rowId)) {
				next.delete(rowId);
			} else {
				next.add(rowId);
			}
			return next;
		});
	};

	const filteredData = () =>
		showSkipped()
			? props.data
			: props.data.filter((job) => !job.SuitabilitySkipped);

	const table = createSolidTable({
		get data() {
			return filteredData();
		},
		columns: props.columns,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		state: {
			get sorting() {
				return sorting();
			},
			get pagination() {
				return pagination();
			},
		},
		onSortingChange: setSorting,
		onPaginationChange: setPagination,
		meta: {
			isExpanded,
			toggleExpanded,
		},
	});

	const pageIndex = () => table.getState().pagination.pageIndex;
	const pageCount = () => table.getPageCount();
	// Data is pre-filtered; use its length for display rather than table's row model count.
	const filteredCount = () => props.data.length;
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

	const filterCount = () => activeFilterCount(props.filters);

	return (
		<div class="flex flex-col gap-3">
			<div class="flex items-center gap-2">
				<div class="relative max-w-xs flex-1">
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
						value={props.filters.q}
						onInput={(e) => props.onChange({ q: e.currentTarget.value })}
						class="pr-3 pl-9"
					/>
				</div>
				<Button
					variant="outline"
					size="sm"
					onClick={() => setFiltersOpen(true)}
					class="relative shrink-0"
				>
					<svg
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						aria-hidden="true"
					>
						<line x1="4" y1="6" x2="20" y2="6" />
						<line x1="8" y1="12" x2="16" y2="12" />
						<line x1="11" y1="18" x2="13" y2="18" />
					</svg>
					Filters
					<Show when={filterCount() > 0}>
						<Badge class="ml-0.5 h-4 min-w-4 px-1 text-[10px]">
							{filterCount()}
						</Badge>
					</Show>
				</Button>
				<div class="flex items-center gap-1.5">
					<label class="text-xs font-medium text-muted" for="suitability-min">
						Min suitability
					</label>
					<Input
						id="suitability-min"
						type="number"
						min="0"
						max="100"
						placeholder="—"
						value={suitabilityMin()}
						onInput={(e) => {
							const raw = e.currentTarget.value.trim();
							setSuitabilityMin(raw);
							setPagination((p) => ({ ...p, pageIndex: 0 }));
						}}
						class="w-20"
					/>
				</div>
				<label class="flex cursor-pointer items-center gap-1.5">
					<input
						type="checkbox"
						checked={showSkipped()}
						onChange={(e) => {
							setShowSkipped(e.currentTarget.checked);
							setPagination((p) => ({ ...p, pageIndex: 0 }));
						}}
						class="h-3.5 w-3.5 rounded border-border accent-primary"
					/>
					<span class="text-xs font-medium text-muted">
						Show below-cutoff jobs
					</span>
				</label>
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
									<>
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
										<Show when={isExpanded(row.original.ID)}>
											<tr class="border-b border-border">
												<td colspan={row.getVisibleCells().length} class="p-0">
													<JobRowExpander job={row.original} />
												</td>
											</tr>
										</Show>
									</>
								)}
							</For>
						</Show>
					</TableBody>
				</Table>

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

			<JobFiltersDialog
				open={filtersOpen()}
				onOpenChange={setFiltersOpen}
				filters={props.filters}
				onChange={props.onChange}
				sourceOptions={props.sourceOptions}
			/>
		</div>
	);
}
