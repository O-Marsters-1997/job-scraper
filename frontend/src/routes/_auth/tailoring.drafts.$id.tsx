import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, Show } from "solid-js";
import { googleWriteHref } from "@/components/GoogleWriteConsent";
import { Icon } from "@/components/Icon";
import { PdfPreview } from "@/components/PdfPreview";
import { ProvenanceDiff } from "@/components/tailoring/ProvenanceDiff";
import { Button } from "@/components/ui/button";
import { KeptDraftExistsError, keptDraft } from "@/lib/tailoring";
import type { Draft } from "@/types/tailoring";
import { fetchDraftPdf } from "../../api/tailoring";
import { useGoogleStatus } from "../../hooks/useGoogle";
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

const SHOW_CHANGES_KEY = "draft.showChanges";

function readShowChanges(): boolean {
	try {
		return localStorage.getItem(SHOW_CHANGES_KEY) === "1";
	} catch {
		return false;
	}
}

function DraftReview(props: { draft: Draft }) {
	const keep = useKeepDraft();
	const google = useGoogleStatus();
	const discard = useDiscardDraft();
	const jobDrafts = useJobDrafts(() => props.draft.jobId);
	const [blockedByKept, setBlockedByKept] = createSignal(false);
	const [showChanges, setShowChanges] = createSignal(readShowChanges());
	const toggleChanges = () => {
		const next = !showChanges();
		setShowChanges(next);
		try {
			localStorage.setItem(SHOW_CHANGES_KEY, next ? "1" : "0");
		} catch {}
	};
	const changes = () => {
		const { base, content } = props.draft;
		return base && content ? { base, content } : null;
	};
	const hasDoc = () =>
		props.draft.status === "ready" && props.draft.outcome !== "discarded";
	const [pdfRevision, setPdfRevision] = createSignal(0);
	const pdfSource = createMemo(
		() => (hasDoc() ? { id: props.draft.id, rev: pdfRevision() } : undefined),
		undefined,
		{ equals: (a, b) => a?.id === b?.id && a?.rev === b?.rev },
	);
	const pdfURL = usePdfUrl(pdfSource, (s) => fetchDraftPdf(s.id));
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

			<Show
				when={
					props.draft.status === "ready" &&
					props.draft.outcome === null &&
					google.data?.connected &&
					!google.data.canEditDocs
				}
			>
				<div class="mb-4 rounded-xl border border-border bg-surface p-4 text-sm">
					<p class="mb-3 text-foreground">
						Allow FastTrack to edit your CV Doc so a kept draft is added as a
						Tab. Without it, kept drafts land as separate Docs.
					</p>
					<Button
						as="a"
						href={googleWriteHref(`/tailoring/drafts/${props.draft.id}`)}
						variant="outline"
						size="sm"
					>
						Reconnect Google
					</Button>
				</div>
			</Show>

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
						<Show when={changes()}>
							<Button
								variant="outline"
								size="sm"
								class="mb-4"
								aria-pressed={showChanges()}
								onClick={toggleChanges}
							>
								Show changes
							</Button>
						</Show>
						<ProvenanceDiff
							draftId={props.draft.id}
							editable={hasDoc()}
							onSaved={() => setPdfRevision((n) => n + 1)}
							provenance={props.draft.provenance}
							findings={props.draft.findings}
							changes={showChanges() ? changes() : null}
						/>
					</div>
				</div>
			</Show>
		</>
	);
}
