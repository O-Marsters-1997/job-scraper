import { z } from "zod";

export const submitJobUrlResultSchema = z.object({
	status: z.enum(["held", "queued"]),
	jobId: z.string().optional(),
});

export type SubmitJobUrlResult = z.infer<typeof submitJobUrlResultSchema>;
