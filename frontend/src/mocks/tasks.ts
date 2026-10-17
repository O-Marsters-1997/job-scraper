import { faker } from "@faker-js/faker";
import {
	type Task,
	type TaskPage,
	type TaskType,
	TaskTypeNameTakenError,
} from "@/types/task";
import { failIfRequested } from "./helpers";
import { getJobs } from "./jobs";

const TYPES = [
	{ ID: "task-type-apply", Name: "Apply", Colour: "#6366f1" },
	{ ID: "task-type-follow-up", Name: "Follow up", Colour: "#c2410c" },
	{ ID: "task-type-reach-out", Name: "Reach out", Colour: "#15803d" },
] as const;

const seedCreatedAt = new Date(Date.now() - 86_400_000).toISOString();

let types: Omit<TaskType, "TaskCount">[] = [
	...TYPES,
	{ ID: "task-type-prep", Name: "Prep", Colour: "#64748b" },
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
			JobID: job?.ID ?? null,
			JobTitle: job?.Title ?? "",
			JobCompanySlug: job?.CompanySlug ?? "",
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
	types = types.filter((t) => t.ID !== id);
	tasks = allTasks().filter((t) => t.TypeID !== id);
}

export function getTaskPage(page: number, pageSize: number): TaskPage {
	const open = allTasks()
		.filter((t) => t.CompletedAt === null)
		.sort((a, b) => b.CreatedAt.localeCompare(a.CreatedAt));
	return {
		items: open.slice((page - 1) * pageSize, page * pageSize),
		total: open.length,
	};
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
