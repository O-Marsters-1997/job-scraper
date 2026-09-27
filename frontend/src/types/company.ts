import { z } from "zod";

export const companySchema = z.object({
	ID: z.string(),
	Slug: z.string(),
	Name: z.string(),
	ATSSource: z.string(),
	ATSToken: z.string(),
	FirstSeenAt: z.string(),
	JobCount: z.number(),
	Tracked: z.boolean(),
	TargetID: z.string(),
	CheckIntervalMinutes: z.number(),
	LastCheckedAt: z.string().nullable(),
});

export type Company = z.infer<typeof companySchema>;

export const companyTrackingSchema = z.object({
	CompanyID: z.string(),
	UserID: z.string(),
	Enabled: z.boolean(),
	CheckIntervalMinutes: z.number(),
});

export type CompanyTracking = z.infer<typeof companyTrackingSchema>;

export const companyBoardSchema = z.object({
	ID: z.string(),
	CompanyID: z.string(),
	Source: z.string(),
	BoardToken: z.string(),
	Status: z.enum(["candidate", "verified", "retired"]),
	VerificationMethod: z.string(),
	VerifiedAt: z.string().nullable(),
	LastLinkedAt: z.string().nullable(),
	RetiredAt: z.string().nullable(),
	LastCompletedAt: z.string().nullable(),
	CreatedAt: z.string(),
});

export type CompanyBoard = z.infer<typeof companyBoardSchema>;

export interface AddCompanyPayload {
	url: string;
	track?: boolean;
}
