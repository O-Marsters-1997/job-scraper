import { z } from "zod";

export const meSchema = z.object({
	id: z.string(),
	username: z.string(),
});

export type Me = z.infer<typeof meSchema>;
