import type { LinkedDraft } from "../types/application";

export type CvStage =
	| "none"
	| "generating"
	| "inProgress"
	| "completing"
	| "complete"
	| "failed";

export const CV_STAGE_LABEL: Record<CvStage, string> = {
	none: "No CV",
	generating: "Generating…",
	inProgress: "In progress",
	completing: "Completing…",
	complete: "Complete",
	failed: "Failed",
};

export function cvStage(draft: LinkedDraft | null): CvStage {
	if (!draft) return "none";
	if (draft.Outcome === "kept") return "complete";
	switch (draft.Status) {
		case "keeping":
			return "completing";
		case "ready":
			return "inProgress";
		case "failed":
			return "failed";
		case "pending":
		case "running":
			return "generating";
	}
}

export function isCvStageBusy(stage: CvStage): boolean {
	return stage === "generating" || stage === "completing";
}
