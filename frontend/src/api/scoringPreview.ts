import type { Preferences } from "../types/scoringConfig";
import {
	type ScoringPreview,
	scoringPreviewSchema,
} from "../types/scoringPreview";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function previewScoring(
	preferences: Preferences,
): Promise<ScoringPreview> {
	return mocked(
		(db) => db.previewScoring(),
		() =>
			apiFetch(
				"/scoring-preview",
				scoringPreviewSchema,
				jsonInit("POST", { preferences }),
			),
	);
}
