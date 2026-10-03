import { createSignal, Show } from "solid-js";
import { useMarkJobsSeen } from "./useJobs";

const TOAST_MS = 8000;

const [seenJobIds, setSeenJobIds] = createSignal<string[] | null>(null);
let hideTimer: ReturnType<typeof setTimeout> | undefined;

export function announceBulkSeen(jobIds: string[]) {
	clearTimeout(hideTimer);
	setSeenJobIds(jobIds);
	hideTimer = setTimeout(() => setSeenJobIds(null), TOAST_MS);
}

export function SeenToast() {
	const markSeen = useMarkJobsSeen();
	const undo = async (jobIds: string[]) => {
		try {
			await markSeen.mutateAsync({ jobIds, seen: false });
		} finally {
			setSeenJobIds(null);
		}
	};
	return (
		<Show when={seenJobIds()}>
			{(ids) => (
				<output class="fixed inset-x-4 bottom-20 z-50 flex items-center gap-3 rounded-lg border border-border-strong bg-surface px-4 py-3 text-sm text-foreground shadow-md md:bottom-4 md:left-auto md:max-w-sm">
					<span class="truncate">Marked {ids().length} jobs as seen.</span>
					<button
						type="button"
						onClick={() => undo(ids())}
						class="ml-auto shrink-0 rounded text-xs font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					>
						Undo
					</button>
				</output>
			)}
		</Show>
	);
}
