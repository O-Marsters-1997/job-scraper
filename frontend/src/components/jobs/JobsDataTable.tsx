import { For, Show } from "solid-js";
import { createSignal } from "solid-js";
import {
  createSolidTable,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type ColumnDef,
  type PaginationState,
  type SortingState,
} from "@tanstack/solid-table";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface JobsDataTableProps<TData> {
  columns: ColumnDef<TData, unknown>[];
  data: TData[];
}

export function JobsDataTable<TData>(props: JobsDataTableProps<TData>) {
  const [globalFilter, setGlobalFilter] = createSignal("");
  const [sorting, setSorting] = createSignal<SortingState>([]);
  const [pagination, setPagination] = createSignal<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
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

  return (
    <div class="space-y-4">
      {/* Search */}
      <input
        type="search"
        placeholder="Search jobs…"
        value={globalFilter()}
        onInput={(e) => {
          setGlobalFilter(e.currentTarget.value);
          setPagination((p) => ({ ...p, pageIndex: 0 }));
        }}
        class="w-full max-w-sm rounded-lg border border-[var(--line)] bg-white/70 px-3 py-2 text-sm text-[var(--sea-ink)] placeholder:text-[var(--sea-ink-soft)] focus:outline-none focus:ring-2 focus:ring-[var(--lagoon)] focus:ring-offset-0"
      />

      {/* Table */}
      <div class="island-shell rounded-xl overflow-hidden">
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
                            <span class="text-[var(--sea-ink-soft)]">
                              {header.column.getIsSorted() === "asc"
                                ? " ↑"
                                : header.column.getIsSorted() === "desc"
                                  ? " ↓"
                                  : " ↕"}
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
                    class="py-12 text-center text-sm text-[var(--sea-ink-soft)]"
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
      </div>

      {/* Pagination */}
      <Show when={pageCount() > 1}>
        <div class="flex flex-col items-center gap-3">
          <p class="text-xs text-[var(--sea-ink-soft)]">
            Showing {start()}–{end()} of {filteredCount()} jobs
          </p>
          <div class="flex items-center gap-2">
            <button
              type="button"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
              class={`inline-flex items-center rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                !table.getCanPreviousPage()
                  ? "pointer-events-none border-[var(--line)] text-[var(--sea-ink-soft)] opacity-40"
                  : "border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] text-[var(--lagoon-deep)] hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
              }`}
            >
              ← Prev
            </button>
            <span class="text-sm text-[var(--sea-ink-soft)]">
              Page {pageIndex() + 1} of {pageCount()}
            </span>
            <button
              type="button"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
              class={`inline-flex items-center rounded-full border px-3 py-1.5 text-sm font-medium transition ${
                !table.getCanNextPage()
                  ? "pointer-events-none border-[var(--line)] text-[var(--sea-ink-soft)] opacity-40"
                  : "border-[rgba(50,143,151,0.3)] bg-[rgba(79,184,178,0.14)] text-[var(--lagoon-deep)] hover:-translate-y-0.5 hover:bg-[rgba(79,184,178,0.24)]"
              }`}
            >
              Next →
            </button>
          </div>
        </div>
      </Show>
    </div>
  );
}
