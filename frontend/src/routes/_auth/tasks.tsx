import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { Pager } from "@/components/Pager";
import { QueryBoundary } from "@/components/QueryBoundary";
import { StatusBadge } from "@/components/StatusBadge";
import { Card } from "@/components/ui/card";
import { formatRelative } from "@/lib/datetime";
import { taskHeadline } from "@/lib/taskHeadline";
import { useSetTaskDone, useTaskPages } from "../../hooks/useTasks";

export const Route = createFileRoute("/_auth/tasks")({
	validateSearch: (search: Record<string, unknown>) => {
		const page = Number(search.page);
		return { page: Number.isInteger(page) && page > 1 ? page : undefined };
	},
	component: TasksPage,
});

function TasksPage() {
	const search = Route.useSearch();
	const navigate = useNavigate({ from: "/tasks" });
	const table = useTaskPages(search, (patch) =>
		navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
	);
	const setDone = useSetTaskDone();

	return (
		<div class="px-7 py-6">
			<PageHeading title="Tasks" subtitle="Things to do on your job search" />

			<QueryBoundary query={table.query} fallbackRows={5}>
				{(data) => (
					<Show
						when={data().items.length > 0}
						fallback={
							<Card class="p-10 text-center">
								<p class="text-sm text-muted">Nothing left to do.</p>
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
												aria-label={`Mark done: ${taskHeadline(task)}`}
												class="size-4 shrink-0 cursor-pointer accent-primary"
												checked={false}
												disabled={setDone.isPending}
												onChange={() =>
													setDone.mutate({ id: task.ID, done: true })
												}
											/>
											<div class="min-w-0 flex-1">
												<p class="truncate text-sm font-medium text-foreground">
													{taskHeadline(task)}
												</p>
												<Show when={task.JobCompanySlug}>
													<p class="text-xs text-faint">
														<Link
															to="/jobs/$id"
															params={{ id: task.JobID ?? "" }}
															class="hover:underline"
														>
															{task.JobCompanySlug}
														</Link>
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
