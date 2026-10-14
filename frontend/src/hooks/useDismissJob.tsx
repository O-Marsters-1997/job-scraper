import { createSignal, Show } from "solid-js";
import { recordEvent } from "@/api/events";
import { GradeChips } from "@/components/jobs/GradeChips";
import { undoPlan } from "@/lib/gradeBatch";
import type { Grade, GradeReason } from "@/types/grade";
import { useExcludeCompanies, useUnexcludeCompanies } from "./useCompanies";
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

interface Exclusion {
	companies: { companyId: string; name: string; added: boolean }[];
	skipped: number;
}

const [exclusion, setExclusion] = createSignal<Exclusion | null>(null);
const [notice, setNotice] = createSignal<string | null>(null);
let noticeTimer: ReturnType<typeof setTimeout> | undefined;

export function announceNotice(text: string) {
	clearTimeout(noticeTimer);
	setNotice(text);
	noticeTimer = setTimeout(() => setNotice(null), TOAST_MS);
}

export function companyIdsOf(jobs: { CompanyID?: string | undefined }[]) {
	const ids = [
		...new Set(jobs.flatMap((j) => (j.CompanyID ? [j.CompanyID] : []))),
	];
	return { ids, skipped: jobs.filter((j) => !j.CompanyID).length };
}

function pluralCompanies(n: number) {
	return n === 1 ? "1 company" : `${n} companies`;
}

function skippedSuffix(skipped: number) {
	return skipped > 0 ? ` Skipped ${skipped} without a company.` : "";
}

function describeExclusion(e: Exclusion) {
	const [only] = e.companies;
	const head =
		e.companies.length === 1 && only
			? `Excluded “${only.name}”.`
			: `Excluded ${pluralCompanies(e.companies.length)}.`;
	return head + skippedSuffix(e.skipped);
}

export function describeKind(
	updated: number,
	kind: "agency" | "direct",
	skipped: number,
) {
	const label = kind === "agency" ? "recruiters" : "direct employers";
	return `Marked ${pluralCompanies(updated)} as ${label}.${skippedSuffix(skipped)}`;
}
const [dismissal, setDismissal] = createSignal<Dismissal | null>(null);
const [bulkGrading, setBulkGrading] = createSignal<BulkGrading | null>(null);
let hideTimer: ReturnType<typeof setTimeout> | undefined;

function showFor(next: Dismissal | null) {
	clearTimeout(hideTimer);
	setBulkGrading(null);
	setExclusion(null);
	setDismissal(next);
	if (next) hideTimer = setTimeout(() => setDismissal(null), TOAST_MS);
}

export function announceBulkGrading(next: BulkGrading) {
	clearTimeout(hideTimer);
	setDismissal(null);
	setExclusion(null);
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
			recordEvent({ type: "job_dismissed", subject_id: job.ID });
			showFor({ jobId: job.ID, title: job.Title, reasons: [] });
		},
		undo: async (jobId: string) => {
			await clear.mutateAsync(jobId);
			recordEvent({ type: "job_dismissed", subject_id: jobId, undone: true });
			showFor(null);
		},
		setReasons: async (jobId: string, reasons: GradeReason[]) => {
			await set.mutateAsync({ jobId, grade: "no", reasons });
			const current = dismissal();
			if (current) showFor({ ...current, reasons });
		},
	};
}

function showExclusion(next: Exclusion | null) {
	clearTimeout(hideTimer);
	setDismissal(null);
	setBulkGrading(null);
	setExclusion(next);
	if (next) hideTimer = setTimeout(() => setExclusion(null), TOAST_MS);
}

export function useExcludeJobCompanies() {
	const exclude = useExcludeCompanies();
	return {
		exclude: async (jobs: { CompanyID?: string | undefined }[]) => {
			const { ids, skipped } = companyIdsOf(jobs);
			if (ids.length === 0) {
				announceNotice(`No companies to exclude.${skippedSuffix(skipped)}`);
				return;
			}
			try {
				const result = await exclude.mutateAsync(ids);
				showExclusion({ companies: result.companies, skipped });
			} catch {
				announceNotice("Could not exclude companies.");
			}
		},
	};
}

export function useExcludeJobCompany() {
	const { exclude } = useExcludeJobCompanies();
	return {
		exclude: (job: { CompanyID?: string | undefined }) => exclude([job]),
	};
}

function ExcludeCompanyToast() {
	const unexclude = useUnexcludeCompanies();
	const undo = async (e: Exclusion) => {
		try {
			await unexclude.mutateAsync(
				e.companies.map((c) => ({
					companyId: c.companyId,
					removeName: c.added,
				})),
			);
		} finally {
			setExclusion(null);
		}
	};
	return (
		<Show when={exclusion()}>
			{(e) => (
				<output class="fixed inset-x-4 bottom-20 z-50 flex items-center gap-3 rounded-lg border border-border-strong bg-surface px-4 py-3 text-sm text-foreground shadow-md md:bottom-4 md:left-auto md:max-w-sm">
					<span class="truncate">{describeExclusion(e())}</span>
					<button
						type="button"
						onClick={() => undo(e())}
						class="ml-auto shrink-0 rounded text-xs font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
					>
						Undo
					</button>
				</output>
			)}
		</Show>
	);
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
			<ExcludeCompanyToast />
			<Show when={notice()}>
				{(text) => (
					<output class="fixed inset-x-4 bottom-20 z-50 rounded-lg border border-border-strong bg-surface px-4 py-3 text-sm text-foreground shadow-md md:bottom-4 md:left-auto md:max-w-sm">
						{text()}
					</output>
				)}
			</Show>
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
