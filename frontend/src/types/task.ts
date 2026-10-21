import { z } from "zod";

export const taskSortSchema = z.enum(["created", "job_posted"]);
export const taskTypeSchema = z.object({
	ID: z.string(),
	Name: z.string(),
	Colour: z.string(),
	TaskCount: z.number().int(),
	CreatedAt: z.string(),
});
export const taskSchema = z.object({
	ID: z.string(),
	TypeID: z.string(),
	TypeName: z.string(),
	TypeColour: z.string(),
	JobID: z.string().nullable(),
	JobTitle: z.string(),
	JobCompanySlug: z.string(),
	JobCompanyID: z.string().nullable(),
	ApplicationID: z.string().nullable(),
	JobPostedAt: z.string().nullable(),
	Link: z.string(),
	Notes: z.string(),
	CompletedAt: z.string().nullable(),
	CreatedAt: z.string(),
	UpdatedAt: z.string(),
});
export const taskPageSchema = z.object({
	items: z.array(taskSchema),
	total: z.number().int(),
});

export type TaskSort = z.infer<typeof taskSortSchema>;
export type TaskType = z.infer<typeof taskTypeSchema>;
export type Task = z.infer<typeof taskSchema>;
export type TaskPage = z.infer<typeof taskPageSchema>;

export class TaskTypeNameTakenError extends Error {
	constructor() {
		super("A task type with that name already exists.");
	}
}
