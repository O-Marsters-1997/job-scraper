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
	ReviewState: z.enum(["", "new", "kept", "dismissed"]),
	TargetID: z.string(),
	CheckIntervalMinutes: z.number(),
	LastCheckedAt: z.string().nullable(),
});

export type Company = z.infer<typeof companySchema>;

export const companyPageSchema = z.object({
	items: z.array(companySchema),
	next_cursor: z.string(),
});

export type CompanyPage = z.infer<typeof companyPageSchema>;

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

const trackedBoardSchema = z.object({
	id: z.string(),
	source: z.string(),
	board_token: z.string(),
	status: z.enum(["candidate", "verified", "retired"]),
	url: z.string(),
});

export const trackedCompanySchema = z.object({
	id: z.string(),
	name: z.string(),
	slug: z.string(),
	enabled: z.boolean(),
	review_state: z.enum(["new", "kept", "dismissed"]),
	check_interval_minutes: z.number(),
	boards: z.array(trackedBoardSchema),
	open_jobs: z.number(),
	relevant_jobs: z.number(),
	last_checked_at: z.string().nullable(),
});

const companyProfileEntrySchema = z.object({
	dimension: z.string(),
	label: z.string(),
	yes: z.number(),
	known: z.number(),
	total: z.number(),
});

export const companyProfileSchema = z.object({
	sectors: z.array(z.string()).nullable(),
	size: z.string(),
	growth: z.string(),
	funding_total: z.string(),
	funding_rounds: z.number(),
	investors: z.array(z.string()).nullable(),
	hq: z.string(),
	hybrid_note: z.string(),
	uk_visa: z.string(),
	glassdoor: z.string(),
	mission: z.string(),
});

export type CompanyProfile = z.infer<typeof companyProfileSchema>;

export const newCompanySchema = z.object({
	id: z.string(),
	name: z.string(),
	slug: z.string(),
	boards: z.array(trackedBoardSchema),
	matching_roles: z.number(),
	best_suitability: z.number().nullable(),
	rollup: z.array(companyProfileEntrySchema),
	profile: companyProfileSchema.nullable(),
});

export type CompanyProfileEntry = z.infer<typeof companyProfileEntrySchema>;
export type NewCompany = z.infer<typeof newCompanySchema>;
export type TrackedBoard = z.infer<typeof trackedBoardSchema>;
export type TrackedCompany = z.infer<typeof trackedCompanySchema>;

export type ReviewState = "new" | "kept" | "dismissed";

export interface AddCompanyPayload {
	url: string;
	track?: boolean;
}
