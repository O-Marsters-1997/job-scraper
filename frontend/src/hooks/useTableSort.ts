import { batch, createSignal } from "solid-js";

type SortDir = "asc" | "desc";

export function useTableSort<K extends string>(initialKey: K) {
	const [sortKey, setSortKey] = createSignal<K>(initialKey);
	const [sortDir, setSortDir] = createSignal<SortDir>("asc");

	function handleSort(key: K) {
		if (sortKey() === key) {
			setSortDir((d) => (d === "asc" ? "desc" : "asc"));
		} else {
			batch(() => {
				setSortKey(() => key);
				setSortDir("asc");
			});
		}
	}

	function sortIcon(key: K): string | null {
		if (sortKey() !== key) return null;
		return sortDir() === "asc" ? "↑" : "↓";
	}

	return { sortKey, sortDir, handleSort, sortIcon };
}
