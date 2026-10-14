import {
	createQuery,
	keepPreviousData,
	type QueryKey,
} from "@tanstack/solid-query";
import type { RowSelectionState } from "@tanstack/solid-table";
import { createEffect, createSignal, on } from "solid-js";
import { clampPage, filterSignature, nextSelection } from "@/lib/serverTable";

interface ServerTableOptions<
	S extends { page?: number | undefined },
	P extends { items: unknown[]; total: number },
> {
	state: () => S;
	onChange: (patch: Partial<S>) => void;
	pageSize: number;
	queryKey: (state: S) => QueryKey;
	queryFn: (state: S) => Promise<P>;
	rowId: (item: P["items"][number]) => string;
}

export function useServerTable<
	S extends { page?: number | undefined },
	P extends { items: unknown[]; total: number },
>(opts: ServerTableOptions<S, P>) {
	type Item = P["items"][number];

	const query = createQuery(() => ({
		queryKey: opts.queryKey(opts.state()),
		queryFn: () => opts.queryFn(opts.state()),
		placeholderData: keepPreviousData,
	}));

	const page = () => opts.state().page ?? 1;
	const items = (): Item[] => query.data?.items ?? [];
	const total = () => query.data?.total ?? 0;

	const setPage = (next: number) =>
		opts.onChange({ page: next > 1 ? next : undefined } as Partial<S>);
	const setFilters = (patch: Partial<S>) =>
		opts.onChange({ page: undefined, ...patch } as Partial<S>);

	const [selected, setSelected] = createSignal<Record<string, Item>>({});
	const rowSelection = (): RowSelectionState =>
		Object.fromEntries(Object.keys(selected()).map((id) => [id, true]));
	const select = (next: RowSelectionState) =>
		setSelected(nextSelection(selected(), next, items(), opts.rowId));
	const clearSelection = () => setSelected({});

	createEffect(
		on(() => filterSignature(opts.state()), clearSelection, { defer: true }),
	);

	createEffect(() => {
		const count = query.data?.total;
		if (count === undefined || query.isPlaceholderData) return;
		const last = clampPage(page(), count, opts.pageSize);
		if (last !== page()) setPage(last);
	});

	return {
		query,
		page,
		pageSize: opts.pageSize,
		items,
		total,
		setPage,
		setFilters,
		rowSelection,
		selectedItems: () => Object.values(selected()),
		select,
		clearSelection,
	};
}
