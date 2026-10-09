import { z } from "zod";

export const meSchema = z.object({
	id: z.string(),
	username: z.string(),
	isAdmin: z.boolean(),
});

export type Me = z.infer<typeof meSchema>;
