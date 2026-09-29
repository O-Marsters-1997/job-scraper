import { createFileRoute, Link } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { PdfPreview } from "@/components/PdfPreview";
import { ProvenanceDiff } from "@/components/tailoring/ProvenanceDiff";
import { Button } from "@/components/ui/button";
import { KeptDraftExistsError, keptDraft } from "@/lib/tailoring";
import type { Draft } from "@/types/tailoring";
import { fetchDraftPdf } from "../../api/tailoring";
import { usePdfUrl } from "../../hooks/usePdfUrl";
import {
	useDiscardDraft,
	useDraft,
	useJobDrafts,
	useKeepDraft,
} from "../../hooks/useTailoring";

export const Route = createFileRoute("/_auth/tailoring/drafts/$id")({
	component: DraftReviewPage,
});

function DraftReviewPage() {
	const params = Route.useParams();
	const draft = useDraft(() => params().id);

	return (
		<div class="px-7 py-6 pb-16">
			<Show
				when={draft.data}
				fallback={
					<p class="text-sm text-muted">
						{draft.isError ? "Draft not found." : "Loading…"}
					</p>
				}
			>
				{(d) => <DraftReview draft={d()} />}
			</Show>
		</div>
	);
}

function DraftReview(props: { draft: Draft }) {
	const keep = useKeepDraft();
	const discard = useDiscardDraft();
	const jobDrafts = useJobDrafts(() => props.draft.jobId);
	const [blockedByKept, setBlockedByKept] = createSignal(false);
	const hasDoc = () =>
		props.draft.status === "ready" && props.draft.outcome !== "discarded";
	const pdfURL = usePdfUrl(
		() => (hasDoc() ? props.draft.id : undefined),
		fetchDraftPdf,
	);
	const otherKept = () => {
		const kept = keptDraft(jobDrafts.data ?? []);
		return kept?.id === props.draft.id ? undefined : kept;
	};

	const openDoc = (url: string | null) => {
		if (url) window.open(url, "_blank", "noopener,noreferrer");
	};
	const doKeep = () =>
		keep.mutate(props.draft.id, {
			onSuccess: (kept) => {
				setBlockedByKept(false);
				openDoc(kept.draftDocUrl);
			},
			onError: (err) => setBlockedByKept(err instanceof KeptDraftExistsError),
		});
	const replaceKept = () => {
		const kept = otherKept();
		if (!kept) return;
		discard.mutate(kept.id, { onSuccess: doKeep });
	};

	return (
		<>
			<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
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
					<p class="mt-0.5 text-xs text-faint">
						A first draft for you to check. Read every bullet against your own
						experience before you use it.
					</p>
				</div>

				<Show when={props.draft.status === "ready"}>
					<div class="flex flex-wrap items-center gap-2">
						<Show when={props.draft.outcome === "kept"}>
							<span class="text-sm text-muted">Kept</span>
						</Show>
						<Show when={props.draft.outcome === "discarded"}>
							<span class="text-sm text-muted">Discarded</span>
						</Show>
						<Show when={props.draft.draftDocUrl}>
							{(url) => (
								<Button
									as="a"
									href={url()}
									target="_blank"
									rel="noopener noreferrer"
									variant="outline"
									size="sm"
								>
									Open in Google Docs
								</Button>
							)}
						</Show>
						<Show when={props.draft.outcome !== "discarded"}>
							<Button
								variant="outline"
								size="sm"
								disabled={discard.isPending}
								onClick={() => discard.mutate(props.draft.id)}
							>
								Discard
							</Button>
						</Show>
						<Show when={props.draft.outcome === null}>
							<Button size="sm" disabled={keep.isPending} onClick={doKeep}>
								Keep draft
							</Button>
						</Show>
					</div>
				</Show>
			</div>

			<Show when={blockedByKept()}>
				<div
					role="alert"
					class="mb-4 rounded-xl border border-border bg-surface p-4 text-sm"
				>
					<p class="mb-3 text-foreground">
						This job already has a kept draft. A job keeps one draft at a time.
					</p>
					<div class="flex flex-wrap gap-2">
						<Show when={otherKept()?.id}>
							{(id) => (
								<Link
									to="/tailoring/drafts/$id"
									params={{ id: id() }}
									class="inline-flex h-8 items-center rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
								>
									View kept draft
								</Link>
							)}
						</Show>
						<Button
							size="sm"
							disabled={!otherKept() || discard.isPending || keep.isPending}
							onClick={replaceKept}
						>
							Discard the kept draft and keep this one
						</Button>
					</div>
				</div>
			</Show>

			<Show when={keep.isError && !blockedByKept()}>
				<p role="alert" class="mb-4 text-sm text-destructive-strong">
					Could not keep the draft. Try again.
				</p>
			</Show>
			<Show when={discard.isError}>
				<p role="alert" class="mb-4 text-sm text-destructive-strong">
					Could not discard the draft. Try again.
				</p>
			</Show>

			<Show
				when={props.draft.status === "ready"}
				fallback={
					<Show
						when={props.draft.status === "failed"}
						fallback={
							<p class="text-sm text-muted" aria-live="polite">
								Writing your draft. This usually takes under a minute.
							</p>
						}
					>
						<p class="text-sm text-destructive-strong">
							Generating the draft failed
							{props.draft.lastError ? `: ${props.draft.lastError}` : "."}
						</p>
					</Show>
				}
			>
				<div class="grid items-start gap-6 lg:grid-cols-2">
					<div class="min-w-0">
						<Show
							when={hasDoc()}
							fallback={
								<p class="rounded-xl border border-border bg-surface p-6 text-sm text-muted">
									This draft was discarded and its Google Doc deleted.
								</p>
							}
						>
							<PdfPreview url={pdfURL} title="Draft CV PDF" />
						</Show>
					</div>
					<div class="min-w-0">
						<ProvenanceDiff
							provenance={props.draft.provenance}
							findings={props.draft.findings}
						/>
					</div>
				</div>
			</Show>
		</>
	);
}
