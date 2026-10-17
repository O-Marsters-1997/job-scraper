import {
	type Task,
	type TaskPage,
	taskPageSchema,
	taskSchema,
} from "../types/task";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export const TASKS_PAGE_SIZE = 25;

export async function fetchOpenTasks(
	page: number,
	pageSize: number,
): Promise<TaskPage> {
	return mocked(
		(db) => db.getTaskPage(page, pageSize),
		() => {
			const query = new URLSearchParams({
				done: "false",
				page: String(page),
				page_size: String(pageSize),
			});
			return apiFetch(`/tasks/?${query}`, taskPageSchema);
		},
	);
}

export async function setTaskDone(id: string, done: boolean): Promise<Task> {
	return mocked(
		(db) => db.setTaskDone(id, done),
		() => apiFetch(`/tasks/${id}/done`, taskSchema, jsonInit("PUT", { done })),
	);
}
