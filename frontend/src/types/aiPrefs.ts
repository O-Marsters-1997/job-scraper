import { z } from "zod";

export const aiPrefsSchema = z.object({
	configuredProviders: z.array(z.string()),
	scoringEnabled: z.boolean(),
});

export type AiPrefs = z.infer<typeof aiPrefsSchema>;
