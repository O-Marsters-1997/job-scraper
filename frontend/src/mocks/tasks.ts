import { faker } from "@faker-js/faker";
import type { Task, TaskPage, TaskType } from "@/types/task";
import { ApiError } from "../api/client";
import type { CreateTaskInput } from "../api/tasks";
import { failIfRequested } from "./helpers";
import { getJobs } from "./jobs";

const TYPES = [
	{ ID: "task-type-apply", Name: "Apply", Colour: "#6366f1" },
	{ ID: "task-type-follow-up", Name: "Follow up", Colour: "#c2410c" },
	{ ID: "task-type-reach-out", Name: "Reach out", Colour: "#15803d" },
] as const;

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

export function getTaskTypes(): TaskType[] {
	return TYPES.map((t) => ({ ...t, CreatedAt: new Date(0).toISOString() }));
}

export function createTask(input: CreateTaskInput): Task {
	failIfRequested("createTask");
	const type = TYPES.find((t) => t.ID === input.type_id);
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
		JobID: job?.ID ?? null,
		JobTitle: job?.Title ?? "",
		JobCompanySlug: job?.CompanySlug ?? "",
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

export function getTaskPage(
	page: number,
	pageSize: number,
	jobId?: string,
): TaskPage {
	const open = allTasks()
		.filter((t) => t.CompletedAt === null)
		.filter((t) => !jobId || t.JobID === jobId)
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
