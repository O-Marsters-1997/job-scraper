import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { PositionCard } from "@/components/experience/PositionCard";
import { PositionForm } from "@/components/experience/PositionForm";
import { Icon } from "@/components/Icon";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import {
	experienceQueryOptions,
	useCreatePosition,
	useExperience,
} from "../../hooks/useExperience";
import { queryClient } from "../../lib/queryClient";

export const Route = createFileRoute("/_auth/experience")({
	loader: () => queryClient.ensureQueryData(experienceQueryOptions),
	component: ExperiencePage,
});

function ExperiencePage() {
	const query = useExperience();
	const create = useCreatePosition();
	const [adding, setAdding] = createSignal(false);

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Experience"
				subtitle="Your Positions and Achievements, the source for tailored CVs"
			>
				<Button class="shrink-0" onClick={() => setAdding(true)}>
					<Icon name="plus" size={12} strokeWidth={2.5} />
					Add position
				</Button>
			</PageHeading>

			<Show when={adding()}>
				<div class="mb-4 rounded-xl border border-border bg-surface p-4">
					<PositionForm
						submitLabel="Add position"
						pending={create.isPending}
						error={create.error?.message}
						onCancel={() => setAdding(false)}
						onSubmit={(input) =>
							create.mutate(input, { onSuccess: () => setAdding(false) })
						}
					/>
				</div>
			</Show>

			<QueryBoundary query={query}>
				{(positions) => (
					<Show
						when={positions().length > 0}
						fallback={
							<Show when={!adding()}>
								<div class="rounded-xl border border-border bg-surface p-10 text-center">
									<p class="text-sm text-muted">
										No positions yet. Add one to start building your experience
										bank.
									</p>
								</div>
							</Show>
						}
					>
						<div class="space-y-4">
							<For each={positions()}>
								{(position, i) => (
									<PositionCard
										position={position}
										index={i()}
										positionIds={positions().map((p) => p.id)}
									/>
								)}
							</For>
						</div>
					</Show>
				)}
			</QueryBoundary>
		</div>
	);
}
