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
export const outreachKindSchema = z.enum([
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

export const contactCompanySchema = z.object({
	id: z.string(),
	name: z.string(),
	kind: z.string(),
	tracked: z.boolean(),
	open_jobs: z.array(
		z.object({
			job_id: z.string(),
			title: z.string(),
			has_contact: z.boolean(),
		}),
	),
	application_ids: z.array(z.string()),
});

export type ContactCompany = z.infer<typeof contactCompanySchema>;

export type RelationshipStatus =
	| "in_conversation"
	| "chase_due"
	| "waiting"
	| "parked"
	| "not_started"
	| "gone_cold"
	| "no_one_yet";

export const relationshipLabels: Record<RelationshipStatus, string> = {
	in_conversation: "In conversation",
	chase_due: "Chase due",
	waiting: "Waiting",
	parked: "Parked",
	not_started: "Not started",
	gone_cold: "Gone cold",
	no_one_yet: "No one yet",
};

export function relationshipStatus(
	contacts: Contact[],
	today: string,
): RelationshipStatus {
	if (contacts.length === 0) return "no_one_yet";
	const has = (test: (c: Contact) => boolean) => contacts.some(test);
	if (has((c) => c.outcome === "interested" || c.outcome === "referred")) {
		return "in_conversation";
	}
	if (
		has(
			(c) =>
				c.stage !== "closed" && !!c.follow_up_by && c.follow_up_by <= today,
		)
	) {
		return "chase_due";
	}
	if (has((c) => c.stage === "messaged")) return "waiting";
	if (has((c) => c.outcome === "not_now")) return "parked";
	if (has((c) => c.stage === "to_contact")) return "not_started";
	return "gone_cold";
}
