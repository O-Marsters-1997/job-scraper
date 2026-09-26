import { z } from "zod";
import { apiFetch } from "./client";
import { useMocks } from "./config";

export const dimensionSpecSchema = z.object({
	key: z.enum(["tech", "role", "domain", "seniority", "work", "stage"]),
	kind: z.enum(["pair", "multi"]),
	stances: z.array(z.string()),
});

export const scoringOptionSchema = z.object({
	id: z.string(),
	dimension: z.enum(["tech", "role", "domain", "seniority", "work", "stage"]),
	label: z.string(),
});

export const scoringOptionsSchema = z.object({
	dimensions: z.array(dimensionSpecSchema),
	options: z.array(scoringOptionSchema),
});

export type DimensionSpec = z.infer<typeof dimensionSpecSchema>;
export type ScoringOption = z.infer<typeof scoringOptionSchema>;
export type ScoringOptionsView = z.infer<typeof scoringOptionsSchema>;

export async function fetchScoringOptions(): Promise<ScoringOptionsView> {
	if (useMocks()) {
		const { getScoringOptions } = await import("../mocks/db");
		return getScoringOptions();
	}
	return apiFetch("/scoring-options", undefined, scoringOptionsSchema);
}
