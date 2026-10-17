import {
	type Task,
	type TaskPage,
	type TaskSort,
	type TaskType,
	taskPageSchema,
	taskSchema,
	taskTypeSchema,
} from "../types/task";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export const TASKS_PAGE_SIZE = 25;

export interface TasksQuery {
	page: number;
	pageSize: number;
	typeId?: string | undefined;
	sort?: TaskSort | undefined;
	done?: boolean | undefined;
}

export async function fetchTasks(q: TasksQuery): Promise<TaskPage> {
	return mocked(
		(db) => db.getTaskPage(q),
		() => {
			const query = new URLSearchParams({
				done: String(q.done ?? false),
				page: String(q.page),
				page_size: String(q.pageSize),
			});
			if (q.typeId) query.set("type_id", q.typeId);
			if (q.sort) query.set("sort", q.sort);
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

export async function setTaskDone(id: string, done: boolean): Promise<Task> {
	return mocked(
		(db) => db.setTaskDone(id, done),
		() => apiFetch(`/tasks/${id}/done`, taskSchema, jsonInit("PUT", { done })),
	);
}
