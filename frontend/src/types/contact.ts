import { z } from "zod";

const contactStageSchema = z.enum([
	"to_contact",
	"messaged",
	"replied",
	"closed",
]);
const contactOutcomeSchema = z.enum([
	"interested",
	"referred",
	"not_now",
	"no_thanks",
	"no_reply",
]);
const contactChannelSchema = z.enum(["linkedin", "email", "call", "inbound"]);
const outreachKindSchema = z.enum([
	"recruiter",
	"hiring_manager",
	"referral",
	"follow_up",
]);
const contactEventKindSchema = z.enum(["added", "sent", "reply", "closed"]);

const contactEventSchema = z.object({
	id: z.string(),
	kind: contactEventKindSchema,
	occurred_on: z.string(),
	channel: contactChannelSchema.optional(),
	template_kind: outreachKindSchema.optional(),
	template_name: z.string().optional(),
	outcome: contactOutcomeSchema.optional(),
	note: z.string().optional(),
	job_id: z.string().nullable(),
	job_title: z.string().optional(),
});

export const contactSchema = z.object({
	id: z.string(),
	name: z.string(),
	role: z.string(),
	company_id: z.string().nullable(),
	company_name: z.string(),
	company_kind: z.string(),
	job_id: z.string().nullable(),
	job_title: z.string(),
	application_id: z.string().nullable(),
	linkedin_url: z.string(),
	email: z.string(),
	stage: contactStageSchema,
	outcome: contactOutcomeSchema.optional(),
	follow_up_by: z.string().nullable(),
	reply: z.string(),
	sent_since_reply: z.number().int(),
	history: z.array(contactEventSchema),
	created_at: z.string(),
});

export const contactSentSchema = z.object({
	contact: contactSchema,
	moved_application_chase: z.boolean(),
});

type ContactOutcome = z.infer<typeof contactOutcomeSchema>;
type OutreachKind = z.infer<typeof outreachKindSchema>;
export type ContactChannel = z.infer<typeof contactChannelSchema>;
export type ContactEvent = z.infer<typeof contactEventSchema>;
export type Contact = z.infer<typeof contactSchema>;
export type ContactSent = z.infer<typeof contactSentSchema>;

export interface ContactInput {
	name: string;
	role: string;
	company_id: string | null;
	company_name: string;
	job_id: string | null;
	linkedin_url: string;
	email: string;
}

export interface MarkSentInput {
	channel: ContactChannel;
	template_kind: OutreachKind | "";
	template_name: string;
}

export type ReplyOutcome = Exclude<ContactOutcome, "no_reply">;

export interface RecordReplyInput {
	outcome: ReplyOutcome;
	reply: string;
}

export const channelLabels: Record<ContactChannel, string> = {
	linkedin: "LinkedIn",
	email: "Email",
	call: "Call",
	inbound: "Inbound",
};

export const replyOutcomes: {
	key: ReplyOutcome;
	label: string;
	effect: string;
}[] = [
	{
		key: "interested",
		label: "Interested",
		effect: "Moves to In conversation.",
	},
	{
		key: "referred",
		label: "Referred me on",
		effect: "Moves to In conversation.",
	},
	{
		key: "not_now",
		label: "Not now",
		effect: "Parks them and brings them back in 90 days.",
	},
	{ key: "no_thanks", label: "No thanks", effect: "Closes the contact." },
];
