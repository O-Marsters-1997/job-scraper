import { z } from "zod";

export const achievementSchema = z.object({
	id: z.string(),
	positionId: z.string(),
	text: z.string(),
});

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
