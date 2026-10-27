import type { ApplicationWithDetails } from "../types/application";
import type { ApplicationStatus } from "../types/applicationStatus";
import { cvStage } from "./cvStage";

export type Selection = Record<string, true>;

export function nextSelection(
	ids: string[],
	selection: Selection,
	anchor: string | undefined,
	id: string,
	extendRange: boolean,
): Selection {
	const from = anchor === undefined ? -1 : ids.indexOf(anchor);
	const to = ids.indexOf(id);
	const next = { ...selection };
	if (extendRange && from !== -1 && to !== -1) {
		for (const rangeId of ids.slice(
			Math.min(from, to),
			Math.max(from, to) + 1,
		)) {
			next[rangeId] = true;
		}
	} else if (next[id]) {
		delete next[id];
	} else {
		next[id] = true;
	}
	return next;
}

export function selectedDraftApps(
	apps: ApplicationWithDetails[],
	selection: Selection,
	statuses: ApplicationStatus[],
): ApplicationWithDetails[] {
	const draftIds = new Set(
		statuses.filter((s) => s.SystemKey === "draft").map((s) => s.ID),
	);
	return apps.filter((a) => selection[a.ID] && draftIds.has(a.StatusID));
}

export function needsCv(app: ApplicationWithDetails): boolean {
	const stage = cvStage(app.Draft);
	return stage === "none" || stage === "failed";
}
