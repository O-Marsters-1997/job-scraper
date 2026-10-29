import { z } from "zod";
import { customOptionSchema } from "./customOption";
import { dimensionKeySchema } from "./dimension";
import { stanceSchema } from "./scoringConfig";

const dimensionSpecSchema = z.object({
	key: dimensionKeySchema,
	kind: z.enum(["pair", "multi", "ladder"]),
	stances: z.array(stanceSchema),
});

export type DimensionSpec = z.infer<typeof dimensionSpecSchema>;

const scoringOptionSchema = z.object({
	id: z.string(),
	dimension: dimensionKeySchema,
	label: z.string(),
	level: z.number().int().optional(),
});

export type ScoringOption = z.infer<typeof scoringOptionSchema>;

export const scoringOptionsSchema = z.object({
	dimensions: z.array(dimensionSpecSchema),
	options: z.array(scoringOptionSchema),
	custom: z.array(customOptionSchema),
	customCap: z.number().int(),
});

export type ScoringOptionsView = z.infer<typeof scoringOptionsSchema>;
