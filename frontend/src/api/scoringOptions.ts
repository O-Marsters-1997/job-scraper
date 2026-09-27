import {
	type ScoringOptionsView,
	scoringOptionsSchema,
} from "../types/scoringOptions";
import { apiFetch } from "./client";
import { useMocks } from "./config";

export async function fetchScoringOptions(): Promise<ScoringOptionsView> {
	if (useMocks()) {
		const { getScoringOptions } = await import("../mocks/db");
		return getScoringOptions();
	}
	return apiFetch("/scoring-options", undefined, scoringOptionsSchema);
}
