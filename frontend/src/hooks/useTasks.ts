import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createTask,
	createTaskType,
	deleteTaskType,
	fetchTasks,
	fetchTaskTypes,
	setTaskDone,
	TASKS_PAGE_SIZE,
	updateTask,
	updateTaskType,
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

export const taskPageQuery = (s: TaskListState) => ({
	queryKey: keys.tasks.list(s),
	queryFn: () =>
		fetchTasks({
			page: s.page ?? 1,
			pageSize: TASKS_PAGE_SIZE,
			typeId: s.type,
			sort: s.sort,
			done: s.done,
		}),
});

export function useTaskPages(
	state: () => TaskListState,
	onChange: (patch: Partial<TaskListState>) => void,
) {
	return useServerTable({
		state,
		onChange,
		pageSize: TASKS_PAGE_SIZE,
		query: taskPageQuery,
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

export const taskTypesQueryOptions = queryOptions({
	queryKey: keys.tasks.types,
	queryFn: fetchTaskTypes,
});

const JOB_TASKS_LIMIT = 50;

export const jobOpenTasksQueryOptions = (jobId: string) =>
	queryOptions({
		queryKey: keys.tasks.forJob(jobId),
		queryFn: () => fetchTasks({ page: 1, pageSize: JOB_TASKS_LIMIT, jobId }),
	});

export function useJobOpenTasks(jobId: () => string) {
	return createQuery(() => jobOpenTasksQueryOptions(jobId()));
}

export function useTaskTypes() {
	return createQuery(() => taskTypesQueryOptions);
}

const TASK_TYPE_INVALIDATES = [keys.tasks.all];

export function useCreateTaskType() {
	return useInvalidatingMutation(
		({ name, colour }: { name: string; colour: string }) =>
			createTaskType(name, colour),
		TASK_TYPE_INVALIDATES,
	);
}

export function useUpdateTaskType() {
	return useInvalidatingMutation(
		({ id, name, colour }: { id: string; name: string; colour: string }) =>
			updateTaskType(id, name, colour),
		TASK_TYPE_INVALIDATES,
	);
}

export function useDeleteTaskType() {
	return useInvalidatingMutation(deleteTaskType, TASK_TYPE_INVALIDATES);
}

export function useCreateTask() {
	return useInvalidatingMutation(createTask, [keys.tasks.all]);
}

export function useUpdateTask() {
	return useInvalidatingMutation(updateTask, [keys.tasks.all]);
}

export function useSetTaskDone() {
	return useInvalidatingMutation(
		({ id, done }: { id: string; done: boolean }) => setTaskDone(id, done),
		[keys.tasks.all],
	);
}
