import { z } from "zod";
import { outreachKindSchema } from "./contact";
import { letterFindingSchema } from "./coverLetter";

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

export const outreachDraftSchema = z.object({
	templateId: z.string(),
	kind: outreachKindSchema,
	body: z.string(),
	achievementId: z.string(),
	hasHookSlot: z.boolean(),
	hook: z.string(),
	hookFindings: z.array(letterFindingSchema),
	cost: z.number(),
});

export type OutreachDraft = z.infer<typeof outreachDraftSchema>;

export interface OutreachDraftQuery {
	templateId?: string | undefined;
	achievementId?: string | undefined;
}

export interface OutreachHookInput {
	achievementId?: string | undefined;
}

export const draftWordLimit = 50;

export function wordCount(text: string): number {
	return text.trim().split(/\s+/).filter(Boolean).length;
}
