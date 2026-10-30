import { z } from "zod";
import { outreachKindSchema } from "./contact";

export const outreachTemplateSchema = z.object({
	id: z.string(),
	kind: outreachKindSchema,
	name: z.string(),
	body: z.string(),
	isDefault: z.boolean(),
});

export type OutreachTemplate = z.infer<typeof outreachTemplateSchema>;
export type OutreachKind = OutreachTemplate["kind"];

export interface OutreachTemplateInput {
	kind: OutreachKind;
	name: string;
	body: string;
}

export const outreachKinds: { key: OutreachKind; label: string }[] = [
	{ key: "recruiter", label: "Recruiter" },
	{ key: "hiring_manager", label: "Hiring manager" },
	{ key: "referral", label: "Referral" },
	{ key: "follow_up", label: "Follow-up" },
];

export const mergeFields = [
	"{first_name}",
	"{role}",
	"{company}",
	"{achievement}",
	"{hook}",
];
