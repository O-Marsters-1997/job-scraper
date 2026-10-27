import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { firstPolishable, polishable } from "@/lib/applicationSelection";
import { CV_STAGE_LABEL, cvStage } from "@/lib/cvStage";
import { loadedData } from "@/lib/queryData";
import { cn } from "@/lib/utils";
import { useApplications } from "../../hooks/useApplications";
import type { ApplicationWithDetails } from "../../types/application";

export function PolishDock(props: { queue: string[]; draftId: string }) {
	const query = useApplications();
	const apps = () => {
		const byId = new Map((loadedData(query) ?? []).map((a) => [a.ID, a]));
		return props.queue.flatMap((id) => byId.get(id) ?? []);
	};
	const index = () => apps().findIndex((a) => a.Draft?.ID === props.draftId);
	const kept = () => apps()[index()]?.Draft?.Outcome === "kept";
	const complete = () => apps().filter((a) => cvStage(a.Draft) === "complete");

	const step = (label: string, target: ApplicationWithDetails | undefined) => (
		<Show
			when={target}
			fallback={
				<Button variant="ghost" size="sm" disabled>
					{label}
				</Button>
			}
		>
			{(app) => (
				<Link
					to="/tailoring/drafts/$id"
					params={{ id: app().Draft?.ID ?? "" }}
					search={{ queue: props.queue }}
					class="inline-flex h-8 items-center rounded-md px-3 text-sm font-medium text-primary transition-colors hover:bg-accent-subtle"
				>
					{label}
				</Link>
			)}
		</Show>
	);

	return (
		<div class="fixed inset-x-0 bottom-0 z-[60] flex flex-wrap items-center gap-3 border-t border-border bg-surface px-4 py-2 shadow-lg">
			{step("← Prev", firstPolishable(apps(), index(), -1))}
			<ul class="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
				<For each={apps()}>
					{(app) => {
						const stage = () => cvStage(app.Draft);
						const chip = () =>
							cn(
								"flex max-w-48 items-center gap-1.5 rounded-full border border-border px-2.5 py-1 text-xs",
								app.Draft?.ID === props.draftId
									? "bg-accent-subtle text-foreground"
									: "text-muted",
								stage() === "failed" && "text-destructive-strong",
							);
						const label = () => (
							<>
								<span class="truncate">{app.JobTitle}</span>
								<span class="shrink-0">{CV_STAGE_LABEL[stage()]}</span>
							</>
						);
						return (
							<li>
								<Show
									when={polishable(app)}
									fallback={<span class={chip()}>{label()}</span>}
								>
									<Link
										to="/tailoring/drafts/$id"
										params={{ id: app.Draft?.ID ?? "" }}
										search={{ queue: props.queue }}
										class={cn(chip(), "hover:bg-accent-subtle")}
									>
										{label()}
									</Link>
								</Show>
							</li>
						);
					}}
				</For>
			</ul>
			<span class="text-xs text-muted" aria-live="polite">
				{kept()
					? "Linked on the application · Next draft →"
					: "Polish, then Keep draft to complete"}
			</span>
			{step("Next →", firstPolishable(apps(), index(), 1))}
			<Show when={complete().length > 0}>
				<Link
					to="/applications/$id"
					params={{ id: complete()[0]?.ID ?? "" }}
					search={{ run: complete().map((a) => a.ID) }}
					class="inline-flex h-8 items-center rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground"
				>
					Send {complete().length} complete
				</Link>
			</Show>
		</div>
	);
}
