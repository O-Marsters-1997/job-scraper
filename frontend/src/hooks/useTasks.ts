import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createTask,
	fetchOpenTasks,
	fetchTaskTypes,
	setTaskDone,
	TASKS_PAGE_SIZE,
} from "../api/tasks";
import { useInvalidatingMutation } from "./useInvalidatingMutation";
import { useServerTable } from "./useServerTable";

interface TaskListState {
	page?: number | undefined;
}

export function useTaskPages(
	state: () => TaskListState,
	onChange: (patch: Partial<TaskListState>) => void,
) {
	return useServerTable({
		state,
		onChange,
		pageSize: TASKS_PAGE_SIZE,
		queryKey: (s) => keys.tasks.list(s.page ?? 1),
		queryFn: (s) => fetchOpenTasks(s.page ?? 1, TASKS_PAGE_SIZE),
		rowId: (task) => task.ID,
	});
}

const openTaskCountQueryOptions = queryOptions({
	queryKey: keys.tasks.count,
	queryFn: async () => (await fetchOpenTasks(1, 1)).total,
});

export function useOpenTaskCount() {
	return createQuery(() => openTaskCountQueryOptions);
}

const JOB_TASKS_LIMIT = 50;

export function useJobOpenTasks(jobId: () => string) {
	return createQuery(() => ({
		queryKey: keys.tasks.forJob(jobId()),
		queryFn: () => fetchOpenTasks(1, JOB_TASKS_LIMIT, jobId()),
	}));
}

export function useTaskTypes() {
	return createQuery(() => ({
		queryKey: keys.tasks.types,
		queryFn: fetchTaskTypes,
	}));
}

export function useCreateTask() {
	return useInvalidatingMutation(createTask, [keys.tasks.all]);
}

export function useSetTaskDone() {
	return useInvalidatingMutation(
		({ id, done }: { id: string; done: boolean }) => setTaskDone(id, done),
		[keys.tasks.all],
	);
}
