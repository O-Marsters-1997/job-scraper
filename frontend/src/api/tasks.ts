import {
	type Task,
	type TaskPage,
	type TaskType,
	taskPageSchema,
	taskSchema,
	taskTypeSchema,
} from "../types/task";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export const TASKS_PAGE_SIZE = 25;

export async function fetchOpenTasks(
	page: number,
	pageSize: number,
	jobId?: string,
): Promise<TaskPage> {
	return mocked(
		(db) => db.getTaskPage(page, pageSize, jobId),
		() => {
			const query = new URLSearchParams({
				done: "false",
				page: String(page),
				page_size: String(pageSize),
			});
			if (jobId) query.set("job_id", jobId);
			return apiFetch(`/tasks/?${query}`, taskPageSchema);
		},
	);
}

export async function fetchTaskTypes(): Promise<TaskType[]> {
	return mocked(
		(db) => db.getTaskTypes(),
		() => apiFetch("/task-types", taskTypeSchema.array()),
	);
}

export interface CreateTaskInput {
	type_id: string;
	job_id: string | null;
	link: string;
	notes: string;
}

export async function createTask(input: CreateTaskInput): Promise<Task> {
	return mocked(
		(db) => db.createTask(input),
		() => apiFetch("/tasks/", taskSchema, jsonInit("POST", input)),
	);
}

export async function setTaskDone(id: string, done: boolean): Promise<Task> {
	return mocked(
		(db) => db.setTaskDone(id, done),
		() => apiFetch(`/tasks/${id}/done`, taskSchema, jsonInit("PUT", { done })),
	);
}
