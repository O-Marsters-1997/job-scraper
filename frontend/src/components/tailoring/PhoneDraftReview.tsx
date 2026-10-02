import { Link } from "@tanstack/solid-router";
import { Match, Show, Switch } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import type { Draft } from "@/types/tailoring";
import { createKeepFlow } from "../../hooks/useKeepFlow";
import { useSaveDraftSlots } from "../../hooks/useTailoring";
import { ChangesDiff } from "./ChangesDiff";
import { KeepNotices } from "./KeepFlow";

const OUTCOME_LABEL = { kept: "Kept", discarded: "Discarded" };

function Status(props: { draft: Draft }) {
	return (
		<Switch>
			<Match when={props.draft.status === "failed"}>
				<div role="alert" class="flex flex-col items-start gap-3">
					<p class="text-sm text-destructive-strong">
						Generating the draft failed
						{props.draft.lastError ? `: ${props.draft.lastError}` : "."}
					</p>
					<Link
						to="/jobs/$id/tailor"
						params={{ id: props.draft.jobId }}
						class="inline-flex h-9 items-center rounded-md border border-border bg-surface px-3 text-sm font-medium text-foreground transition-colors hover:bg-surface-muted"
					>
						Try again
					</Link>
				</div>
			</Match>
			<Match when={props.draft.status !== "ready" || !props.draft.content}>
				<p class="text-sm text-muted" aria-live="polite">
					Tailoring…
				</p>
			</Match>
		</Switch>
	);
}

function Review(props: {
	draft: Draft;
	base: NonNullable<Draft["base"]>;
	content: NonNullable<Draft["content"]>;
}) {
	const flow = createKeepFlow(
		() => props.draft,
		async () => true,
	);
	const save = useSaveDraftSlots(() => props.draft.id);
	const open = () => props.draft.status === "ready" && !props.draft.outcome;

	return (
		<>
			<KeepNotices draft={props.draft} editable={false} flow={flow} />
			<Show when={save.isError}>
				<p role="alert" class="text-sm text-destructive-strong">
					Could not undo that change. Try again.
				</p>
			</Show>
			<ChangesDiff
				base={props.base}
				content={props.content}
				provenance={props.draft.provenance}
				onUndo={
					open() ? (slotId, text) => save.mutate([{ slotId, text }]) : undefined
				}
			/>
			<div class="sticky bottom-0 -mx-4 border-t border-border bg-surface px-4 py-3">
				<Show
					when={props.draft.outcome ?? undefined}
					fallback={
						<Button
							class="w-full"
							disabled={
								!open() || save.isPending || flow.keepMutation.isPending
							}
							onClick={() => flow.keep()}
						>
							{props.draft.status === "keeping" ? "Keeping…" : "Keep"}
						</Button>
					}
				>
					{(outcome) => (
						<p class="text-center text-sm text-muted">
							{OUTCOME_LABEL[outcome()]}
						</p>
					)}
				</Show>
			</div>
		</>
	);
}

export function PhoneDraftReview(props: { draft: Draft }) {
	return (
		<div class="flex min-h-full flex-col gap-4 px-4 py-4">
			<div>
				<Link
					to="/jobs/$id"
					params={{ id: props.draft.jobId }}
					class="mb-1 flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
				>
					<Icon name="chevronLeft" size={14} />
					Job
				</Link>
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Draft CV
				</h1>
			</div>
			<Status draft={props.draft} />
			<Show
				when={
					props.draft.status !== "failed" &&
					props.draft.base &&
					props.draft.content
						? { base: props.draft.base, content: props.draft.content }
						: undefined
				}
			>
				{(c) => (
					<Review draft={props.draft} base={c().base} content={c().content} />
				)}
			</Show>
		</div>
	);
}
