import { z } from "zod";

export const scoreRowSchema = z.object({
	key: z.string(),
	label: z.string(),
	stance: z.string(),
	resolved: z.enum(["yes", "no", "unknown", "retired"]),
	effect: z.enum(["meets", "misses", "unknown", "neutral", "retired"]),
	overridden: z.boolean(),
});

export type ScoreRow = z.infer<typeof scoreRowSchema>;

export const jobSchema = z.object({
	ID: z.string(),
	Title: z.string(),
	Location: z.string(),
	URL: z.string(),
	CompanySlug: z.string(),
	CompanyID: z.string().optional(),
	Source: z.string(),
	UpdatedAt: z.string(),
	ScrapedAt: z.string(),
	DaysInOffice: z.number().nullable().optional(),
	WorkArrangement: z.string().optional(),
	SalaryRaw: z.string().optional(),
	SuitabilityScore: z.number().nullable(),
	Breakdown: z.array(scoreRowSchema).nullable().optional(),
	// Optional rich fields — populated in demo mode; absent from the live backend
	Description: z.string().optional(),
	Skills: z.array(z.string()).optional(),
	EmploymentType: z.string().optional(),
	ExperienceLevel: z.string().optional(),
	TeamName: z.string().optional(),
	CompanySize: z.string().optional(),
	SalaryRange: z.string().optional(),
});

export type Job = z.infer<typeof jobSchema>;

export const jobPageSchema = z.object({
	items: z.array(jobSchema),
	next_cursor: z.string(),
});

export type JobPage = z.infer<typeof jobPageSchema>;
