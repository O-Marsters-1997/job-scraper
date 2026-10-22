import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Draft } from "@/types/tailoring";

export type ChecklistStepKey = "cv" | "coverLetter" | "apply" | "followUp";

export interface ChecklistInput {
	statusSystemKey: ApplicationStatus["SystemKey"];
	hasKeptDraft: boolean;
	hasDoneLetter: boolean;
	applyTaskOpen: boolean;
	chaseBy: string | null;
}

export function checklistDone(
	input: ChecklistInput,
): Record<ChecklistStepKey, boolean> {
	const pastDraft = input.statusSystemKey !== "draft";
	return {
		cv: input.hasKeptDraft || pastDraft,
		coverLetter: input.hasDoneLetter,
		apply: pastDraft && !input.applyTaskOpen,
		followUp: false,
	};
}

// Kept draft first, else the newest draft with no outcome whose status isn't "failed".
// Drafts arrive newest first (ListJobDrafts).
export function linkedDraft(drafts: Draft[]): Draft | undefined {
	return (
		drafts.find((d) => d.outcome === "kept") ??
		drafts.find((d) => !d.outcome && d.status !== "failed")
	);
}
