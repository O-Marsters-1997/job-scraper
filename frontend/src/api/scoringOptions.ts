import {
	type ScoringOptionsView,
	scoringOptionsSchema,
} from "../types/scoringOptions";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchScoringOptions(): Promise<ScoringOptionsView> {
	return mocked(
		(db) => db.getScoringOptions(),
		() => apiFetch("/scoring-options", undefined, scoringOptionsSchema),
	);
}
