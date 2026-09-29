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
