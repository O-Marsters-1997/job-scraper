import { z } from "zod";
import { draftOutcomeSchema, draftStatusSchema } from "./tailoring";

const linkedDraftSchema = z.object({
	ID: z.string(),
	Status: draftStatusSchema,
	Outcome: draftOutcomeSchema.nullable(),
	LastError: z.string(),
});

export type LinkedDraft = z.infer<typeof linkedDraftSchema>;

export const applicationSchema = z.object({
	ID: z.string(),
	UserID: z.string(),
	JobID: z.string(),
	StatusID: z.string(),
	AppliedAt: z.string().nullable(),
	SalaryInfo: z.string(),
	CreatedAt: z.string(),
	UpdatedAt: z.string(),
	ChaseBy: z.string().nullable(),
});

export type Application = z.infer<typeof applicationSchema>;

export const applicationMoveSchema = z.object({
	Application: applicationSchema,
	FilledAppliedAt: z.boolean(),
});

export type ApplicationMove = z.infer<typeof applicationMoveSchema>;

export interface MoveApplicationPayload {
	status_id: string;
	clear_applied_at?: boolean;
}

export const applicationWithDetailsSchema = applicationSchema.extend({
	JobTitle: z.string(),
	JobCompanySlug: z.string(),
	JobLocation: z.string(),
	JobURL: z.string(),
	JobClosedAt: z.string().nullable(),
	StatusName: z.string(),
	StatusColour: z.string(),
	Draft: linkedDraftSchema.nullable(),
});

export type ApplicationWithDetails = z.infer<
	typeof applicationWithDetailsSchema
>;

export interface JobApplicationSummary {
	ApplicationID: string;
	StatusID: string;
	StatusName: string;
	StatusColour: string;
}

export interface CreateApplicationPayload {
	job_id: string;
	status_id?: string | undefined;
	applied_at?: string | null | undefined;
	salary_info?: string | undefined;
}

export type UpdateApplicationPayload = Partial<
	Omit<CreateApplicationPayload, "job_id">
>;
