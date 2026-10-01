import { z } from "zod";

export const googleStatusSchema = z.object({
	connected: z.boolean(),
	email: z.string().optional(),
	canWrite: z.boolean().default(false),
	canEditDocs: z.boolean().default(false),
});

export type GoogleStatus = z.infer<typeof googleStatusSchema>;
