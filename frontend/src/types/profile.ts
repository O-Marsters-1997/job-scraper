import { z } from "zod";

export const profileSchema = z.object({
	username: z.string(),
	email: z.string(),
	cvFilenamePattern: z.string(),
});

export type Profile = z.infer<typeof profileSchema>;
