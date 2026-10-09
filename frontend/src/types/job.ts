import { z } from "zod";
import { GRADES } from "./grade";

export const BANDS = ["great", "good", "fair", "poor"] as const;
export type Band = (typeof BANDS)[number];
export const bandSchema = z.enum(BANDS).or(z.literal(""));

export const scoreRowSchema = z.object({
	key: z.string(),
	label: z.string(),
	stance: z.string(),
	resolved: z.enum(["yes", "no", "unknown", "retired"]),
	effect: z.enum([
		"meets",
		"misses",
		"unknown",
		"neutral",
		"retired",
		"blocked",
		"gated",
		"favourite",
		"level",
	]),
	overridden: z.boolean(),
	corrected: z.boolean().optional(),
});

export type ScoreRow = z.infer<typeof scoreRowSchema>;

export const jobListingSchema = z.object({
	source: z.string(),
	url: z.string(),
	first_seen_at: z.string(),
});

export type JobListing = z.infer<typeof jobListingSchema>;

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
	Band: bandSchema.optional(),
	Grade: z.enum(GRADES).or(z.literal("")).optional(),
	Seen: z.boolean().optional(),
	CompanyFavourite: z.boolean().optional(),
	Breakdown: z.array(scoreRowSchema).nullable().optional(),
	Listings: z.array(jobListingSchema).nullable().optional(),
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
