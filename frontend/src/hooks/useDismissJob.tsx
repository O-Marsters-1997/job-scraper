import { createSignal, Show } from "solid-js";
import { GradeChips } from "@/components/jobs/GradeChips";
import { undoPlan } from "@/lib/gradeBatch";
import type { Grade, GradeReason } from "@/types/grade";
import { useClearGrade, useSetGrade } from "./useGrades";
import { useMarkSeenAfterGrade } from "./useJobs";

const TOAST_MS = 8000;

interface Dismissal {
	jobId: string;
	title: string;
	reasons: GradeReason[];
}

interface BulkGrading {
	jobIds: string[];
	priors: Record<string, Grade | null>;
}

const [dismissal, setDismissal] = createSignal<Dismissal | null>(null);
const [bulkGrading, setBulkGrading] = createSignal<BulkGrading | null>(null);
let hideTimer: ReturnType<typeof setTimeout> | undefined;

function showFor(next: Dismissal | null) {
	clearTimeout(hideTimer);
	setBulkGrading(null);
	setDismissal(next);
	if (next) hideTimer = setTimeout(() => setDismissal(null), TOAST_MS);
}

export function announceBulkGrading(next: BulkGrading) {
	clearTimeout(hideTimer);
	setDismissal(null);
	setBulkGrading(next);
	hideTimer = setTimeout(() => setBulkGrading(null), TOAST_MS);
}

export function useDismissJob() {
	const set = useSetGrade();
	const clear = useClearGrade();
	const markSeen = useMarkSeenAfterGrade();
	return {
		dismiss: async (job: { ID: string; Title: string }) => {
			await set.mutateAsync({ jobId: job.ID, grade: "no", reasons: [] });
			await markSeen([job.ID]);
			showFor({ jobId: job.ID, title: job.Title, reasons: [] });
		},
		undo: async (jobId: string) => {
			await clear.mutateAsync(jobId);
			showFor(null);
		},
		setReasons: async (jobId: string, reasons: GradeReason[]) => {
			await set.mutateAsync({ jobId, grade: "no", reasons });
			const current = dismissal();
			if (current) showFor({ ...current, reasons });
		},
	};
}

function BulkGradeToast() {
	const set = useSetGrade();
	const clear = useClearGrade();
	const undo = async (b: BulkGrading) => {
		const { restore, clear: toClear } = undoPlan(b.jobIds, b.priors);
		try {
			await Promise.all([
				...restore.map((r) => set.mutateAsync(r)),
				...toClear.map((id) => clear.mutateAsync(id)),
			]);
		} finally {
			setBulkGrading(null);
		}
	};
	return (
		<Show when={bulkGrading()}>
			{(b) => (
				<output class="fixed inset-x-4 bottom-20 z-50 flex items-center gap-3 rounded-lg border border-border-strong bg-surface px-4 py-3 text-sm text-foreground shadow-md md:bottom-4 md:left-auto md:max-w-sm">
					<span class="truncate">Graded {b().jobIds.length} jobs.</span>
					<button
						type="button"
						onClick={() => undo(b())}
						class="ml-auto shrink-0 rounded text-xs font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					>
						Undo
					</button>
				</output>
			)}
		</Show>
	);
}

export function DismissToast() {
	const { undo, setReasons } = useDismissJob();
	return (
		<>
			<BulkGradeToast />
			<Show when={dismissal()}>
				{(d) => (
					<output class="fixed inset-x-4 bottom-20 z-50 flex flex-col gap-2 rounded-lg border border-border-strong bg-surface px-4 py-3 text-sm text-foreground shadow-md md:bottom-4 md:left-auto md:max-w-sm">
						<div class="flex items-center gap-3">
							<span class="truncate">Dismissed “{d().title}”.</span>
							<button
								type="button"
								onClick={() => undo(d().jobId)}
								class="ml-auto shrink-0 rounded text-xs font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
							>
								Undo
							</button>
						</div>
						<GradeChips
							selected={d().reasons}
							onChange={(reasons) => setReasons(d().jobId, reasons)}
						/>
					</output>
				)}
			</Show>
		</>
	);
}
