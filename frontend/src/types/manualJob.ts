import { z } from "zod";
import { jobSchema } from "./job";

export const manualJobResultSchema = z.object({
	job: jobSchema,
	existing: z.boolean(),
});

export type ManualJobResult = z.infer<typeof manualJobResultSchema>;

export interface AddManualJobInput {
	url: string;
	title: string;
	company: string;
	description: string;
}
