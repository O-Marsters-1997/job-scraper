import {
	type Task,
	type TaskPage,
	type TaskType,
	TaskTypeNameTakenError,
	taskPageSchema,
	taskSchema,
	taskTypeSchema,
} from "../types/task";
import { apiFetch, apiFetchVoid, jsonInit, rethrowStatus } from "./client";
import { mocked } from "./config";

const nameTaken = rethrowStatus({ 409: () => new TaskTypeNameTakenError() });

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

export async function fetchTaskTypes(): Promise<TaskType[]> {
	return mocked(
		(db) => db.getTaskTypes(),
		() => apiFetch("/task-types", taskTypeSchema.array()),
	);
}

export async function createTaskType(
	name: string,
	colour: string,
): Promise<TaskType> {
	return mocked(
		(db) => db.createTaskType(name, colour),
		() =>
			apiFetch(
				"/task-types",
				taskTypeSchema,
				jsonInit("POST", { name, colour }),
			).catch(nameTaken),
	);
}

export async function updateTaskType(
	id: string,
	name: string,
	colour: string,
): Promise<TaskType> {
	return mocked(
		(db) => db.updateTaskType(id, name, colour),
		() =>
			apiFetch(
				`/task-types/${id}`,
				taskTypeSchema,
				jsonInit("PATCH", { name, colour }),
			).catch(nameTaken),
	);
}

export async function deleteTaskType(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteTaskType(id),
		() => apiFetchVoid(`/task-types/${id}`, { method: "DELETE" }),
	);
}

export async function setTaskDone(id: string, done: boolean): Promise<Task> {
	return mocked(
		(db) => db.setTaskDone(id, done),
		() => apiFetch(`/tasks/${id}/done`, taskSchema, jsonInit("PUT", { done })),
	);
}
