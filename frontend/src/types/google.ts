import { z } from "zod";

export const googleStatusSchema = z.object({
	connected: z.boolean(),
	email: z.string().optional(),
});

export type GoogleStatus = z.infer<typeof googleStatusSchema>;
