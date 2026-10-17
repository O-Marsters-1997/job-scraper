import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createTask,
	fetchTasks,
	fetchTaskTypes,
	setTaskDone,
	TASKS_PAGE_SIZE,
} from "../api/tasks";
import type { TaskSort } from "../types/task";
import { useInvalidatingMutation } from "./useInvalidatingMutation";
import { useServerTable } from "./useServerTable";

export interface TaskListState {
	page?: number | undefined;
	type?: string | undefined;
	sort?: TaskSort | undefined;
	done?: boolean | undefined;
}

export function useTaskPages(
	state: () => TaskListState,
	onChange: (patch: Partial<TaskListState>) => void,
) {
	return useServerTable({
		state,
		onChange,
		pageSize: TASKS_PAGE_SIZE,
		queryKey: (s) => keys.tasks.list(s),
		queryFn: (s) =>
			fetchTasks({
				page: s.page ?? 1,
				pageSize: TASKS_PAGE_SIZE,
				typeId: s.type,
				sort: s.sort,
				done: s.done,
			}),
		rowId: (task) => task.ID,
	});
}

const openTaskCountQueryOptions = queryOptions({
	queryKey: keys.tasks.count,
	queryFn: async () => (await fetchTasks({ page: 1, pageSize: 1 })).total,
});

export function useOpenTaskCount() {
	return createQuery(() => openTaskCountQueryOptions);
}

const JOB_TASKS_LIMIT = 50;

export function useJobOpenTasks(jobId: () => string) {
	return createQuery(() => ({
		queryKey: keys.tasks.forJob(jobId()),
		queryFn: () =>
			fetchTasks({ page: 1, pageSize: JOB_TASKS_LIMIT, jobId: jobId() }),
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
