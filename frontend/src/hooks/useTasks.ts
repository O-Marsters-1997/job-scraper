import { createQuery, queryOptions } from "@tanstack/solid-query";
import { keys } from "../api/keys";
import {
	createTaskType,
	deleteTaskType,
	fetchTasks,
	fetchTaskTypes,
	setTaskDone,
	TASKS_PAGE_SIZE,
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

const taskTypesQueryOptions = queryOptions({
	queryKey: keys.tasks.types,
	queryFn: fetchTaskTypes,
});

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

export function useSetTaskDone() {
	return useInvalidatingMutation(
		({ id, done }: { id: string; done: boolean }) => setTaskDone(id, done),
		[keys.tasks.all],
	);
}
