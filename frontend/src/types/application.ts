import { z } from "zod";

export const applicationSchema = z.object({
	ID: z.string(),
	UserID: z.string(),
	JobID: z.string(),
	StatusID: z.string(),
	Notes: z.string(),
	AppliedAt: z.string().nullable(),
	SalaryInfo: z.string(),
	CreatedAt: z.string(),
	UpdatedAt: z.string(),
	ChaseBy: z.string().nullable(),
});

export type Application = z.infer<typeof applicationSchema>;

export const applicationWithDetailsSchema = applicationSchema.extend({
	JobTitle: z.string(),
	JobCompanySlug: z.string(),
	JobLocation: z.string(),
	JobURL: z.string(),
	JobClosedAt: z.string().nullable(),
	StatusName: z.string(),
	StatusColour: z.string(),
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
	notes?: string | undefined;
	applied_at?: string | null | undefined;
	salary_info?: string | undefined;
}

export type UpdateApplicationPayload = Partial<
	Omit<CreateApplicationPayload, "job_id">
>;
