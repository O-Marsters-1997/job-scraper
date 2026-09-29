import { type QueryKey, useQueryClient } from "@tanstack/solid-query";
import { createSignal, For, onCleanup } from "solid-js";

const UNDO_WINDOW_MS = 5000;

interface PendingDelete {
	id: string;
	label: string;
	verb: string;
}

export function useUndoDelete(opts: {
	commit: (id: string, keepalive: boolean) => Promise<void>;
	queryKey: QueryKey;
	verb: string;
	errorMessage: string;
	onError: (message: string) => void;
}) {
	const queryClient = useQueryClient();
	const [pending, setPending] = createSignal<PendingDelete[]>([]);
	const [inFlight, setInFlight] = createSignal<string[]>([]);
	const timers = new Map<string, ReturnType<typeof setTimeout>>();

	const drop = (id: string) =>
		setPending((list) => list.filter((p) => p.id !== id));

	const commit = async (id: string, keepalive: boolean) => {
		clearTimeout(timers.get(id));
		timers.delete(id);
		drop(id);
		setInFlight((ids) => [...ids, id]);
		try {
			await opts.commit(id, keepalive);
			await queryClient.invalidateQueries({ queryKey: opts.queryKey });
		} catch {
			opts.onError(opts.errorMessage);
		} finally {
			setInFlight((ids) => ids.filter((x) => x !== id));
		}
	};

	const flush = () => {
		for (const { id } of pending()) void commit(id, true);
	};

	window.addEventListener("pagehide", flush);
	onCleanup(() => {
		window.removeEventListener("pagehide", flush);
		flush();
	});

	return {
		isHidden: (id: string) =>
			pending().some((p) => p.id === id) || inFlight().includes(id),
		pending,
		remove: (id: string, label: string) => {
			setPending((list) => [...list, { id, label, verb: opts.verb }]);
			timers.set(
				id,
				setTimeout(() => void commit(id, false), UNDO_WINDOW_MS),
			);
		},
		undo: (id: string) => {
			clearTimeout(timers.get(id));
			timers.delete(id);
			drop(id);
		},
	};
}

export function UndoToasts(props: {
	items: PendingDelete[];
	onUndo: (id: string) => void;
}) {
	return (
		<output class="pointer-events-none fixed inset-x-4 bottom-4 z-50 flex flex-col items-center gap-2 sm:items-start sm:inset-x-auto sm:left-1/2">
			<For each={props.items}>
				{(item) => (
					<div class="pointer-events-auto flex max-w-full items-center gap-3 rounded-lg border border-border-strong bg-surface px-4 py-2.5 text-sm text-foreground shadow-md">
						<span class="truncate">
							{item.verb} “{item.label}”.
						</span>
						<button
							type="button"
							onClick={() => props.onUndo(item.id)}
							class="shrink-0 rounded text-xs font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
						>
							Undo
						</button>
					</div>
				)}
			</For>
		</output>
	);
}
