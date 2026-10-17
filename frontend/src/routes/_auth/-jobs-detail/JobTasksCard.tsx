import { For, Show } from "solid-js";
import { StatusBadge } from "@/components/StatusBadge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useJobOpenTasks, useSetTaskDone } from "@/hooks/useTasks";

export function JobTasksCard(props: { jobId: string; onAdd: () => void }) {
	const tasks = useJobOpenTasks(() => props.jobId);
	const setDone = useSetTaskDone();

	return (
		<Card>
			<CardHeader class="pb-2">
				<CardTitle>Tasks</CardTitle>
			</CardHeader>
			<CardContent class="gap-3">
				<Show
					when={tasks.data?.items.length}
					fallback={<p class="text-xs text-faint">No open tasks</p>}
				>
					<ul class="flex flex-col gap-2">
						<For each={tasks.data?.items}>
							{(task) => (
								<li class="flex items-center gap-2">
									<input
										type="checkbox"
										aria-label={`Mark done: ${task.TypeName}`}
										class="size-4 shrink-0 cursor-pointer accent-primary"
										checked={false}
										disabled={setDone.isPending}
										onChange={() => setDone.mutate({ id: task.ID, done: true })}
									/>
									<StatusBadge name={task.TypeName} colour={task.TypeColour} />
									<Show when={task.Notes}>
										<span class="min-w-0 truncate text-xs text-muted">
											{task.Notes}
										</span>
									</Show>
								</li>
							)}
						</For>
					</ul>
				</Show>
				<Button
					variant="outline"
					size="sm"
					class="w-full"
					onClick={props.onAdd}
				>
					Add task
				</Button>
			</CardContent>
		</Card>
	);
}
