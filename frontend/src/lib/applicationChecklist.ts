import type { ApplicationStatus } from "@/types/applicationStatus";

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
