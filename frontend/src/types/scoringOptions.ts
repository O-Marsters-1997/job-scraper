import { z } from "zod";
import { stanceSchema } from "./scoringConfig";

const dimensionKeySchema = z.enum([
	"tech",
	"role",
	"domain",
	"seniority",
	"work",
	"stage",
	"size",
	"employment",
]);

const dimensionSpecSchema = z.object({
	key: dimensionKeySchema,
	kind: z.enum(["pair", "multi"]),
	stances: z.array(stanceSchema),
});

export type DimensionSpec = z.infer<typeof dimensionSpecSchema>;

const scoringOptionSchema = z.object({
	id: z.string(),
	dimension: dimensionKeySchema,
	label: z.string(),
});

export type ScoringOption = z.infer<typeof scoringOptionSchema>;

export const scoringOptionsSchema = z.object({
	dimensions: z.array(dimensionSpecSchema),
	options: z.array(scoringOptionSchema),
});

export type ScoringOptionsView = z.infer<typeof scoringOptionsSchema>;
