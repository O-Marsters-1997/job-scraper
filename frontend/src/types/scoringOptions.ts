import { z } from "zod";

export const dimensionSpecSchema = z.object({
	key: z.enum(["tech", "role", "domain", "seniority", "work", "stage"]),
	kind: z.enum(["pair", "multi"]),
	stances: z.array(z.string()),
});

export type DimensionSpec = z.infer<typeof dimensionSpecSchema>;

export const scoringOptionSchema = z.object({
	id: z.string(),
	dimension: z.enum(["tech", "role", "domain", "seniority", "work", "stage"]),
	label: z.string(),
});

export type ScoringOption = z.infer<typeof scoringOptionSchema>;

export const scoringOptionsSchema = z.object({
	dimensions: z.array(dimensionSpecSchema),
	options: z.array(scoringOptionSchema),
});

export type ScoringOptionsView = z.infer<typeof scoringOptionsSchema>;
