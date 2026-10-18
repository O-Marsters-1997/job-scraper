import { createFileRoute, Link } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { PageSkeleton } from "@/components/PageSkeleton";
import { QueryBoundary } from "@/components/QueryBoundary";
import { DraftEditor } from "@/components/tailoring/DraftEditor";
import { PhoneDraftReview } from "@/components/tailoring/PhoneDraftReview";
import type { Draft } from "@/types/tailoring";
import { LayoutUnsupportedError } from "../../api/tailoring";
import { useDraft, useDraftLayout } from "../../hooks/useTailoring";

export const Route = createFileRoute("/_auth/tailoring/drafts/$id")({
	component: DraftReviewPage,
});

function DraftReviewPage() {
	const params = Route.useParams();
	const draft = useDraft(() => params().id);

	return (
		<QueryBoundary
			query={draft}
			notFound={<p class="px-7 py-6 text-sm text-muted">Draft not found.</p>}
		>
			{(d) => (
				<>
					<div class="md:hidden">
						<PhoneDraftReview draft={d()} />
					</div>
					<div class="hidden md:contents">
						<Show
							when={d().status === "ready" || d().status === "keeping"}
							fallback={<Progress draft={d()} />}
						>
							<ReviewableDraft draft={d()} />
						</Show>
					</div>
				</>
			)}
		</QueryBoundary>
	);
}

function ReviewableDraft(props: { draft: Draft }) {
	const layout = useDraftLayout(() => props.draft.id);
	const reason = () =>
		layout.error instanceof LayoutUnsupportedError
			? layout.error.message
			: layout.isError
				? "The layout could not be loaded."
				: undefined;
	return (
		<Show when={!layout.isPending} fallback={<PageSkeleton />}>
			<DraftEditor
				draft={props.draft}
				layout={layout.data ?? null}
				layoutError={reason()}
			/>
		</Show>
	);
}

function Progress(props: { draft: Draft }) {
	return (
		<div class="px-7 py-6">
			<Link
				to="/jobs/$id"
				params={{ id: props.draft.jobId }}
				class="mb-1 flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
			>
				<Icon name="chevronLeft" size={14} />
				Job
			</Link>
			<h1 class="text-lg font-bold tracking-tight text-foreground">Draft CV</h1>
			<Show
				when={props.draft.status === "failed"}
				fallback={
					<p class="mt-3 text-sm text-muted" aria-live="polite">
						Writing your draft. This usually takes under a minute.
					</p>
				}
			>
				<p class="mt-3 text-sm text-destructive-strong">
					Generating the draft failed
					{props.draft.lastError ? `: ${props.draft.lastError}` : "."}
				</p>
			</Show>
		</div>
	);
}
