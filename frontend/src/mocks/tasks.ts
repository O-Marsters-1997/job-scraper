import { faker } from "@faker-js/faker";
import type { Task, TaskPage } from "@/types/task";
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
