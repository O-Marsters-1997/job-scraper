import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createTaskType,
	deleteTaskType,
	fetchOpenTasks,
	fetchTaskTypes,
	setTaskDone,
	TASKS_PAGE_SIZE,
	updateTaskType,
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

const taskTypesQueryOptions = queryOptions({
	queryKey: keys.taskTypes,
	queryFn: fetchTaskTypes,
});

export function useTaskTypes() {
	return createQuery(() => taskTypesQueryOptions);
}

const TASK_TYPE_INVALIDATES = [keys.taskTypes, keys.tasks.all];

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

export function useSetTaskDone() {
	return useInvalidatingMutation(
		({ id, done }: { id: string; done: boolean }) => setTaskDone(id, done),
		[keys.tasks.all],
	);
}
