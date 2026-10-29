import { z } from "zod";
import { dimensionKeySchema } from "./dimension";

export const customOptionSchema = z.object({
	id: z.string(),
	dimension: dimensionKeySchema,
	label: z.string(),
	question: z.string(),
});

export const customOptionDraftSchema = z.object({
	question: z.string(),
	dimension: dimensionKeySchema,
	answers: z.array(
		z.object({
			jobId: z.string(),
			title: z.string(),
			company: z.string(),
			resolved: z.enum(["yes", "no", "unknown"]),
		}),
	),
});

export type CustomOption = z.infer<typeof customOptionSchema>;
export type CustomOptionDraft = z.infer<typeof customOptionDraftSchema>;

export type CustomOptionInput = Pick<
	CustomOption,
	"label" | "question" | "dimension"
>;

export type CustomOptionDraftInput = {
	phrase: string;
	question?: string;
	dimension?: CustomOption["dimension"];
};
