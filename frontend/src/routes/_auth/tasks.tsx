import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { Pager } from "@/components/Pager";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { ToggleChip } from "@/components/ToggleChip";
import { CreateTaskDialog } from "@/components/tasks/CreateTaskDialog";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import { formatRelative } from "@/lib/datetime";
import { taskHeadline } from "@/lib/taskHeadline";
import { type Task, taskSortSchema } from "@/types/task";
import {
	type TaskListState,
	taskPageQuery,
	taskTypesQueryOptions,
	useDeleteTask,
	useSetTaskDone,
	useTaskPages,
	useTaskTypes,
} from "../../hooks/useTasks";
import { queryClient } from "../../lib/queryClient";

interface Choice {
	value: string;
	label: string;
}

const SORT_CHOICES: Choice[] = [
	{ value: "created", label: "Newest first" },
	{ value: "job_posted", label: "Job posted" },
];

const ALL_TYPES: Choice = { value: "", label: "All types" };

function parseTaskSearch(search: Record<string, unknown>): TaskListState {
	const page = Number(search.page);
	return {
		page: Number.isInteger(page) && page > 1 ? page : undefined,
		type:
			typeof search.type === "string" && search.type ? search.type : undefined,
		sort: taskSortSchema.safeParse(search.sort).data,
		done: search.done === true || search.done === "true" ? true : undefined,
	};
}

export const Route = createFileRoute("/_auth/tasks")({
	validateSearch: parseTaskSearch,
	loader: ({ location }) =>
		Promise.all([
			queryClient.prefetchQuery(
				taskPageQuery(parseTaskSearch(location.search)),
			),
			queryClient.prefetchQuery(taskTypesQueryOptions),
		]),
	component: TasksPage,
});

function ChoiceSelect(props: {
	label: string;
	options: Choice[];
	value: string;
	onChange: (value: string) => void;
}) {
	return (
		<Select
			options={props.options}
			optionValue={(o) => o?.value ?? ""}
			optionTextValue={(o) => o?.label ?? ""}
			disallowEmptySelection
			value={props.options.find((o) => o.value === props.value)}
			onChange={(opt) => {
				if (opt) props.onChange(opt.value);
			}}
			itemComponent={(itemProps) => (
				<SelectItem item={itemProps.item}>
					<SelectItemLabel>{itemProps.item.rawValue?.label}</SelectItemLabel>
				</SelectItem>
			)}
		>
			<SelectTrigger aria-label={props.label} class="w-44">
				<Select.Value<Choice>>
					{(state) => state.selectedOption()?.label}
				</Select.Value>
			</SelectTrigger>
			<SelectContent />
		</Select>
	);
}

function TasksPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/tasks" });
	const table = useTaskPages(search, (patch) =>
		navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
	);
	const setDone = useSetTaskDone();
	const remove = useDeleteTask();
	const [dialogOpen, setDialogOpen] = createSignal(false);
	const [editing, setEditing] = createSignal<Task | undefined>();
	const openDialog = (task?: Task) => {
		setEditing(task);
		setDialogOpen(true);
	};
	const types = useTaskTypes();
	const typeChoices = () => [
		ALL_TYPES,
		...(types.data ?? []).map((t) => ({ value: t.ID, label: t.Name })),
	];

	return (
		<div class="px-7 py-6">
			<PageHeading title="Tasks" subtitle="Things to do on your job search">
				<Button onClick={() => openDialog()}>New task</Button>
			</PageHeading>
			<CreateTaskDialog
				open={dialogOpen()}
				onOpenChange={setDialogOpen}
				task={editing()}
			/>

			<div class="mb-4 flex flex-wrap items-center gap-2">
				<ChoiceSelect
					label="Filter by type"
					options={typeChoices()}
					value={search().type ?? ""}
					onChange={(value) => table.setFilters({ type: value || undefined })}
				/>
				<ChoiceSelect
					label="Sort tasks"
					options={SORT_CHOICES}
					value={search().sort ?? "created"}
					onChange={(value) =>
						table.setFilters({
							sort:
								value === "created" ? undefined : taskSortSchema.parse(value),
						})
					}
				/>
				<ToggleChip
					active={search().done === true}
					onClick={() =>
						table.setFilters({ done: search().done ? undefined : true })
					}
				>
					Show done
				</ToggleChip>
			</div>

			<QueryBoundary query={table.query} fallbackRows={5}>
				{(data) => (
					<Show
						when={data().items.length > 0}
						fallback={
							<Card class="p-10 text-center">
								<p class="text-sm text-muted">
									{search().done ? "No done tasks." : "Nothing left to do."}
								</p>
							</Card>
						}
					>
						<div class="overflow-hidden rounded-xl border border-border bg-surface">
							<ul class="divide-y divide-border">
								<For each={data().items}>
									{(task) => (
										<li class="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-muted">
											<input
												type="checkbox"
												aria-label={`${task.CompletedAt ? "Mark not done" : "Mark done"}: ${taskHeadline(task)}`}
												class="size-4 shrink-0 cursor-pointer accent-primary"
												checked={task.CompletedAt !== null}
												disabled={setDone.isPending}
												onChange={() =>
													setDone.mutate({
														id: task.ID,
														done: task.CompletedAt === null,
													})
												}
											/>
											<div class="min-w-0 flex-1">
												<p class="truncate text-sm font-medium text-foreground">
													<Show when={task.JobID} fallback={taskHeadline(task)}>
														{(jobId) => (
															<Show
																when={task.ApplicationID}
																fallback={
																	<Link
																		to="/jobs/$id"
																		params={{ id: jobId() }}
																		target="_blank"
																		class="hover:underline"
																	>
																		{taskHeadline(task)}
																	</Link>
																}
															>
																{(applicationId) => (
																	<Link
																		to="/applications/$id"
																		params={{ id: applicationId() }}
																		target="_blank"
																		class="hover:underline"
																	>
																		{taskHeadline(task)}
																	</Link>
																)}
															</Show>
														)}
													</Show>
												</p>
												<Show when={task.JobCompanySlug}>
													<p class="text-xs text-faint">
														<Show
															when={task.JobCompanyID}
															fallback={task.JobCompanySlug}
														>
															{(companyId) => (
																<Link
																	to="/companies/$id"
																	params={{ id: companyId() }}
																	target="_blank"
																	class="hover:underline"
																>
																	{task.JobCompanySlug}
																</Link>
															)}
														</Show>
														<Show when={task.JobPostedAt}>
															{(posted) => (
																<> · posted {formatRelative(posted())}</>
															)}
														</Show>
													</p>
												</Show>
											</div>
											<StatusBadge
												name={task.TypeName}
												colour={task.TypeColour}
											/>
											<Button
												variant="outline"
												size="sm"
												aria-label={`Edit: ${taskHeadline(task)}`}
												onClick={() => openDialog(task)}
											>
												Edit
											</Button>
											<Button
												variant="ghost"
												size="sm"
												aria-label={`Delete: ${taskHeadline(task)}`}
												disabled={remove.isPending}
												onClick={() => remove.mutate(task.ID)}
											>
												Delete
											</Button>
										</li>
									)}
								</For>
							</ul>
							<Pager
								total={data().total}
								page={table.page()}
								pageSize={table.pageSize}
								noun="tasks"
								onPage={table.setPage}
							/>
						</div>
					</Show>
				)}
			</QueryBoundary>
		</div>
	);
}
