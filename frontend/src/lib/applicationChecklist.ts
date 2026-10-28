import type { ApplicationStatus } from "@/types/applicationStatus";
import type { Draft } from "@/types/tailoring";

export type ChecklistStepKey = "cv" | "coverLetter" | "apply" | "followUp";

export interface ChecklistInput {
	statusSystemKey: ApplicationStatus["SystemKey"];
	hasKeptDraft: boolean;
	hasOpenDraft: boolean;
	hasLetter: boolean;
	hasDoneLetter: boolean;
	applyTaskOpen: boolean;
	chaseBy: string | null;
}

export type ChecklistState = "todo" | "inProgress" | "done";

export function checklistState(
	input: ChecklistInput,
): Record<ChecklistStepKey, ChecklistState> {
	const pastDraft = input.statusSystemKey !== "draft";
	return {
		cv:
			input.hasKeptDraft || pastDraft
				? "done"
				: input.hasOpenDraft
					? "inProgress"
					: "todo",
		coverLetter: input.hasDoneLetter
			? "done"
			: input.hasLetter
				? "inProgress"
				: "todo",
		apply: pastDraft && !input.applyTaskOpen ? "done" : "todo",
		followUp: "todo",
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
