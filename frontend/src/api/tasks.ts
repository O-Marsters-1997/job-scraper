import {
	type Task,
	type TaskPage,
	type TaskSort,
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

export interface TasksQuery {
	page: number;
	pageSize: number;
	typeId?: string | undefined;
	sort?: TaskSort | undefined;
	done?: boolean | undefined;
	jobId?: string | undefined;
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
			if (q.jobId) query.set("job_id", q.jobId);
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

export interface UpdateTaskInput {
	id: string;
	type_id: string;
	link: string;
	notes: string;
}

export async function updateTask({
	id,
	...body
}: UpdateTaskInput): Promise<Task> {
	return mocked(
		(db) => db.updateTask(id, body),
		() => apiFetch(`/tasks/${id}`, taskSchema, jsonInit("PATCH", body)),
	);
}

export async function setTaskDone(id: string, done: boolean): Promise<Task> {
	return mocked(
		(db) => db.setTaskDone(id, done),
		() => apiFetch(`/tasks/${id}/done`, taskSchema, jsonInit("PUT", { done })),
	);
}
