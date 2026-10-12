import { z } from "zod";

export const CONTACT_CHANNELS = [
	"linkedin_message",
	"email",
	"call",
	"inbound",
] as const;
export const contactChannelSchema = z.enum(CONTACT_CHANNELS);
export type ContactChannel = z.infer<typeof contactChannelSchema>;

export const CONTACT_OUTCOMES = [
	"no_reply",
	"rejected",
	"spoke",
	"submitted",
] as const;
export const contactOutcomeSchema = z.enum(CONTACT_OUTCOMES);
export type ContactOutcome = z.infer<typeof contactOutcomeSchema>;

export const recruiterContactSchema = z.object({
	id: z.string(),
	job_id: z.string(),
	contacted_on: z.string(),
	channel: contactChannelSchema,
	outcome: contactOutcomeSchema,
	created_at: z.string(),
});
export type RecruiterContact = z.infer<typeof recruiterContactSchema>;

export const recruiterContactSummarySchema = z.object({
	count: z.number(),
	last_contacted_on: z.string(),
	best_outcome: contactOutcomeSchema,
});
export type RecruiterContactSummary = z.infer<
	typeof recruiterContactSummarySchema
>;

export interface LogRecruiterContactPayload {
	jobId: string;
	contactedOn: string;
	channel: ContactChannel;
	outcome: ContactOutcome;
}
