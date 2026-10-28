import { faker } from "@faker-js/faker";
import {
	type Task,
	type TaskPage,
	type TaskType,
	TaskTypeNameTakenError,
} from "@/types/task";
import { ApiError } from "../api/apiError";
import type {
	CreateTaskInput,
	TasksQuery,
	UpdateTaskInput,
} from "../api/tasks";
import { failIfRequested } from "./helpers";
import { getJobs } from "./jobs";

const TYPES = [
	{
		ID: "task-type-apply",
		Name: "Apply",
		Colour: "#6366f1",
		SystemKey: "apply",
	},
	{
		ID: "task-type-follow-up",
		Name: "Follow up",
		Colour: "#c2410c",
		SystemKey: "",
	},
	{
		ID: "task-type-reach-out",
		Name: "Reach out",
		Colour: "#15803d",
		SystemKey: "",
	},
] as const;

const seedCreatedAt = new Date(Date.now() - 86_400_000).toISOString();

let types: Omit<TaskType, "TaskCount">[] = [
	...TYPES,
	{
		ID: "task-type-prep",
		Name: "Prep",
		Colour: "#64748b",
		SystemKey: "" as const,
	},
].map((t) => ({ ...t, CreatedAt: seedCreatedAt }));

function seedTasks(): Task[] {
	const jobs = getJobs();
	return TYPES.map((type, i) => {
		const job = jobs[i];
		const at = new Date(Date.now() - (i + 1) * 3_600_000).toISOString();
		return {
			ID: faker.string.uuid(),
			TypeID: type.ID,
			TypeName: type.Name,
			TypeColour: type.Colour,
			TypeSystemKey: type.SystemKey,
			JobID: job?.ID ?? null,
			JobTitle: job?.Title ?? "",
			JobCompanySlug: job?.CompanySlug ?? "",
			JobCompanyID: job?.CompanyID ?? null,
			ApplicationID: null,
			JobPostedAt: job?.PostedAt ?? null,
			Link: job ? "" : "https://example.com/careers",
			Notes: "",
			CompletedAt: null,
			CreatedAt: at,
			UpdatedAt: at,
		};
	});
}

let tasks: Task[] | undefined;

function allTasks(): Task[] {
	tasks ??= seedTasks();
	return tasks;
}

function withCount(type: Omit<TaskType, "TaskCount">): TaskType {
	return {
		...type,
		TaskCount: allTasks().filter((t) => t.TypeID === type.ID).length,
	};
}

function assertNameFree(name: string, exceptId?: string) {
	if (types.some((t) => t.Name === name && t.ID !== exceptId)) {
		throw new TaskTypeNameTakenError();
	}
}

export function getTaskTypes(): TaskType[] {
	return types.map(withCount);
}

export function createTaskType(name: string, colour: string): TaskType {
	failIfRequested("createTaskType");
	assertNameFree(name);
	const type = {
		ID: faker.string.uuid(),
		Name: name,
		Colour: colour,
		SystemKey: "" as const,
		CreatedAt: new Date().toISOString(),
	};
	types = [...types, type];
	return withCount(type);
}

export function updateTaskType(
	id: string,
	name: string,
	colour: string,
): TaskType {
	const prev = types.find((t) => t.ID === id);
	if (!prev) throw new Error("Task type not found");
	assertNameFree(name, id);
	const updated = { ...prev, Name: name, Colour: colour };
	types = types.map((t) => (t.ID === id ? updated : t));
	tasks = allTasks().map((t) =>
		t.TypeID === id ? { ...t, TypeName: name, TypeColour: colour } : t,
	);
	return withCount(updated);
}

export function deleteTaskType(id: string): void {
	if (types.find((t) => t.ID === id)?.SystemKey) {
		throw new ApiError(400, `/task-types/${id}`, {
			error: "this task type is used by the app and can't be deleted",
		});
	}
	types = types.filter((t) => t.ID !== id);
	tasks = allTasks().filter((t) => t.TypeID !== id);
}

export function createTask(input: CreateTaskInput): Task {
	failIfRequested("createTask");
	const type = types.find((t) => t.ID === input.type_id);
	if (!type) throw new Error("Unknown task type");
	if (input.link && !/^https?:\/\//.test(input.link)) {
		throw new ApiError(400, "/tasks/", {
			error: "link must be an http or https URL",
		});
	}
	const job = getJobs().find((j) => j.ID === input.job_id);
	const now = new Date().toISOString();
	const task: Task = {
		ID: faker.string.uuid(),
		TypeID: type.ID,
		TypeName: type.Name,
		TypeColour: type.Colour,
		TypeSystemKey: type.SystemKey,
		JobID: job?.ID ?? null,
		JobTitle: job?.Title ?? "",
		JobCompanySlug: job?.CompanySlug ?? "",
		JobCompanyID: job?.CompanyID ?? null,
		ApplicationID: null,
		JobPostedAt: job?.PostedAt ?? null,
		Link: input.link,
		Notes: input.notes,
		CompletedAt: null,
		CreatedAt: now,
		UpdatedAt: now,
	};
	tasks = [...allTasks(), task];
	return task;
}

const postedMs = (t: Task) =>
	t.JobPostedAt ? new Date(t.JobPostedAt).getTime() : Number.NEGATIVE_INFINITY;

export function getTaskPage(q: TasksQuery): TaskPage {
	const rows = allTasks()
		.filter((t) => (t.CompletedAt !== null) === (q.done ?? false))
		.filter((t) => !q.typeId || t.TypeID === q.typeId)
		.filter((t) => !q.jobId || t.JobID === q.jobId)
		.sort((a, b) =>
			q.sort === "job_posted"
				? postedMs(b) - postedMs(a)
				: b.CreatedAt.localeCompare(a.CreatedAt),
		);
	return {
		items: rows.slice((q.page - 1) * q.pageSize, q.page * q.pageSize),
		total: rows.length,
	};
}

export function updateTask(
	id: string,
	input: Omit<UpdateTaskInput, "id">,
): Task {
	failIfRequested("updateTask");
	const prev = allTasks().find((t) => t.ID === id);
	if (!prev) throw new Error("Task not found");
	const type = types.find((t) => t.ID === input.type_id);
	if (!type) throw new Error("Unknown task type");
	if (input.link && !/^https?:\/\//.test(input.link)) {
		throw new ApiError(400, `/tasks/${id}`, {
			error: "link must be an http or https URL",
		});
	}
	const updated: Task = {
		...prev,
		TypeID: type.ID,
		TypeName: type.Name,
		TypeColour: type.Colour,
		TypeSystemKey: type.SystemKey,
		Link: input.link,
		Notes: input.notes,
		UpdatedAt: new Date().toISOString(),
	};
	tasks = allTasks().map((t) => (t.ID === id ? updated : t));
	return updated;
}

export function completeOpenApplyTasks(jobId: string): void {
	const now = new Date().toISOString();
	tasks = allTasks().map((t) =>
		t.JobID === jobId && t.TypeSystemKey === "apply" && !t.CompletedAt
			? { ...t, CompletedAt: now, UpdatedAt: now }
			: t,
	);
}

export function deleteTask(id: string): void {
	tasks = allTasks().filter((t) => t.ID !== id);
}

export function setTaskDone(id: string, done: boolean): Task {
	failIfRequested("setTaskDone");
	const prev = allTasks().find((t) => t.ID === id);
	if (!prev) throw new Error("Task not found");
	const now = new Date().toISOString();
	const updated: Task = {
		...prev,
		CompletedAt: done ? (prev.CompletedAt ?? now) : null,
		UpdatedAt: now,
	};
	tasks = allTasks().map((t) => (t.ID === id ? updated : t));
	return updated;
}
