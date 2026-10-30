import { z } from "zod";

export const playgroundQueueSchema = z.object({
	jobIds: z.array(z.string()),
	bands: z.array(
		z.object({
			band: z.enum(["great", "good", "fair", "poor"]),
			reviewed: z.number(),
			remaining: z.number(),
		}),
	),
});
export type PlaygroundQueue = z.infer<typeof playgroundQueueSchema>;
