import { z } from "zod";

const achievementStatsSchema = z.object({
	inCvs: z.number(),
	leftOut: z.number(),
	applied: z.number(),
	replied: z.number(),
});

export const achievementSchema = z.object({
	id: z.string(),
	positionId: z.string(),
	text: z.string(),
	context: z.string(),
	stats: achievementStatsSchema.nullable(),
});

export type AchievementStats = z.infer<typeof achievementStatsSchema>;

export const contextChangeSchema = z.object({
	achievementId: z.string(),
	text: z.string(),
	before: z.string(),
	after: z.string(),
});

export const grillPromptSchema = z.object({ text: z.string() });

export type GrillPrompt = z.infer<typeof grillPromptSchema>;
export type ContextChange = z.infer<typeof contextChangeSchema>;

export const positionSchema = z.object({
	id: z.string(),
	employer: z.string(),
	title: z.string(),
	startDate: z.string().nullable(),
	endDate: z.string().nullable(),
	achievements: z.array(achievementSchema),
});

export type Achievement = z.infer<typeof achievementSchema>;
export type Position = z.infer<typeof positionSchema>;

export interface PositionInput {
	employer: string;
	title: string;
	startDate: string | null;
	endDate: string | null;
}

const importPositionSchema = z.object({
	employer: z.string(),
	title: z.string(),
	startDate: z.string().nullable(),
	endDate: z.string().nullable(),
	achievements: z.array(z.string()),
	employerExists: z.boolean(),
});

const importSkillSchema = z.object({
	name: z.string(),
	category: z.string(),
	exists: z.boolean(),
});

export const importPreviewSchema = z.object({
	positions: z.array(importPositionSchema),
	skills: z.array(importSkillSchema),
});

export type ImportPosition = z.infer<typeof importPositionSchema>;
export type ImportSkill = z.infer<typeof importSkillSchema>;
export type ImportPreview = z.infer<typeof importPreviewSchema>;

export const bankSkillSchema = z.object({
	id: z.string(),
	name: z.string(),
	category: z.string(),
	sortOrder: z.number(),
});

export type BankSkill = z.infer<typeof bankSkillSchema>;

export interface BankSkillInput {
	name: string;
	category: string;
}
