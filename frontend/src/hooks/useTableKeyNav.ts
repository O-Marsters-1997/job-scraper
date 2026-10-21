import { createEffect, createSignal, on } from "solid-js";
import { useShortcuts } from "@/hooks/useShortcuts";

const JUMP = 5;

interface TableKeyNavOptions {
	rowIds: () => string[];
	data: () => unknown;
	page: () => number;
	lastPage: () => number;
	onPage: (page: number) => void;
	rowUrl: (id: string) => string;
	extraShortcuts?: Record<string, (e: KeyboardEvent) => unknown>;
}

function isInteractive(target: EventTarget | null) {
	return (
		target instanceof Element &&
		target.closest("a, button, input, [role=checkbox]") !== null
	);
}

export function useTableKeyNav(opts: TableKeyNavOptions) {
	let container: HTMLElement | undefined;
	let searchInput: HTMLInputElement | undefined;
	let focusAfterPaging = false;
	const [activeRow, setActiveRow] = createSignal<string | null>(null);

	const activateFirstRow = () => {
		if (document.activeElement instanceof HTMLElement) {
			document.activeElement.blur();
		}
		container?.scrollIntoView({ block: "start" });
		setActiveRow(opts.rowIds()[0] ?? null);
	};

	const moveActive = (step: number) => {
		const ids = opts.rowIds();
		const current = activeRow();
		const from = current === null ? -1 : ids.indexOf(current);
		const next =
			from === -1 ? 0 : Math.min(Math.max(from + step, 0), ids.length - 1);
		const id = ids[next];
		if (id === undefined) return;
		setActiveRow(id);
		container
			?.querySelector(`tr[data-row-id="${CSS.escape(id)}"]`)
			?.scrollIntoView({ block: "nearest" });
	};

	const openActive = (e: KeyboardEvent) => {
		const id = activeRow();
		if (id === null || isInteractive(e.target)) return false;
		window.open(opts.rowUrl(id), "_blank", "noopener");
	};

	const goToPage = (page: number) => {
		focusAfterPaging = true;
		opts.onPage(page);
	};
	const nextPage = () => {
		if (opts.page() < opts.lastPage()) goToPage(opts.page() + 1);
	};
	const prevPage = () => {
		if (opts.page() > 1) goToPage(opts.page() - 1);
	};

	createEffect(
		on(
			opts.data,
			() => {
				if (focusAfterPaging) {
					focusAfterPaging = false;
					activateFirstRow();
				} else if (!opts.rowIds().includes(activeRow() ?? "")) {
					setActiveRow(null);
				}
			},
			{ defer: true },
		),
	);

	useShortcuts({
		ArrowDown: (e) => moveActive(e.shiftKey ? JUMP : 1),
		j: () => moveActive(1),
		ArrowUp: (e) => moveActive(e.shiftKey ? -JUMP : -1),
		k: () => moveActive(-1),
		Enter: openActive,
		"/": () => {
			searchInput?.scrollIntoView({ block: "nearest" });
			searchInput?.focus();
			searchInput?.select();
		},
		"]": nextPage,
		"[": prevPage,
		ArrowRight: () => (activeRow() === null ? false : nextPage()),
		ArrowLeft: () => (activeRow() === null ? false : prevPage()),
		...opts.extraShortcuts,
	});

	return {
		activeRow,
		containerRef: (el: HTMLElement) => {
			container = el;
		},
		searchRef: (el: HTMLInputElement) => {
			searchInput = el;
		},
		onSearchKeyDown: (e: KeyboardEvent) => {
			if (e.key !== "Enter" || e.isComposing) return;
			if (opts.rowIds().length === 0) return;
			e.preventDefault();
			activateFirstRow();
		},
		rowProps: (id: string) => ({
			"data-row-id": id,
			"data-active": activeRow() === id ? "" : undefined,
			onClick: (e: MouseEvent) => {
				if (!isInteractive(e.target)) setActiveRow(id);
			},
		}),
	};
}
