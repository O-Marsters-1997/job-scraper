import type { Task } from "@/types/task";

export function taskHeadline(task: Task): string {
	if (task.JobID !== null && task.JobTitle !== "") {
		return `${task.TypeName} · ${task.JobTitle}`;
	}
	const host = linkHost(task.Link);
	if (host) return `${task.TypeName} · ${host}`;
	const firstLine = task.Notes.split("\n", 1)[0]?.trim();
	if (firstLine) return `${task.TypeName} · ${firstLine}`;
	return task.TypeName;
}

function linkHost(link: string): string {
	try {
		return new URL(link).host;
	} catch {
		return "";
	}
}
