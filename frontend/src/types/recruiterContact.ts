import { z } from "zod";

export const contactChannelSchema = z.enum([
	"linkedin_message",
	"email",
	"call",
	"inbound",
]);
export const contactOutcomeSchema = z.enum([
	"no_reply",
	"rejected",
	"spoke",
	"submitted",
]);
export const recruiterContactSchema = z.object({
	id: z.string(),
	job_id: z.string(),
	contacted_on: z.string(),
	channel: contactChannelSchema,
	outcome: contactOutcomeSchema,
	created_at: z.string(),
});

export type ContactChannel = z.infer<typeof contactChannelSchema>;
export type ContactOutcome = z.infer<typeof contactOutcomeSchema>;
export type RecruiterContact = z.infer<typeof recruiterContactSchema>;

export interface LogRecruiterContactPayload {
	contacted_on: string;
	channel: ContactChannel;
	outcome: ContactOutcome;
}

export const channelLabels: Record<ContactChannel, string> = {
	linkedin_message: "LinkedIn message",
	email: "Email",
	call: "Call",
	inbound: "Inbound",
};

export const outcomeLabels: Record<ContactOutcome, string> = {
	no_reply: "No reply",
	rejected: "Rejected",
	spoke: "Spoke",
	submitted: "Submitted",
};

export const recruiterContactSummarySchema = z.object({
	count: z.number(),
	last_contacted_on: z.string(),
	best_outcome: contactOutcomeSchema,
});

export type RecruiterContactSummary = z.infer<
	typeof recruiterContactSummarySchema
>;
