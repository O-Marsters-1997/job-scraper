import { Link } from "@tanstack/solid-router";
import { createSignal, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { isSettled } from "@/lib/tailoring";
import { GoogleWriteConsent } from "../../../components/GoogleWriteConsent";
import { useGoogleStatus } from "../../../hooks/useGoogle";
import {
	type CVRef,
	useCreateDraft,
	useDraft,
} from "../../../hooks/useTailoring";

export function GenerateStep(props: {
	jobId: () => string;
	cv: () => CVRef | undefined;
	achievementIds: () => string[];
	onBack: () => void;
}) {
	const google = useGoogleStatus();
	const create = useCreateDraft();
	const [draftId, setDraftId] = createSignal<string>();
	const draft = useDraft(draftId);

	const generate = () => {
		const ref = props.cv();
		if (!ref) return;
		create.mutate(
			{
				jobId: props.jobId(),
				...ref,
				achievementIds: props.achievementIds(),
			},
			{ onSuccess: (r) => setDraftId(r.id) },
		);
	};
	const retry = () => {
		setDraftId(undefined);
		generate();
	};

	return (
		<div class="space-y-4">
			<GoogleWriteConsent returnTo={`/jobs/${props.jobId()}/tailor`} />
			<Card class="block p-5">
				<Show
					when={draftId()}
					fallback={
						<>
							<p class="mb-4 text-sm text-muted">
								FastTrack copies your CV into a new Google Doc and rewrites the
								bullets of the roles you mapped, using only the{" "}
								{props.achievementIds().length} Achievements you chose. Treat
								the result as a first draft.
							</p>
							<Show when={create.error}>
								<p class="mb-3 text-sm text-danger">
									Could not start the draft. Check your selection and try again.
								</p>
							</Show>
							<div class="flex gap-2">
								<Button variant="outline" onClick={props.onBack}>
									Back
								</Button>
								<Button
									disabled={
										create.isPending ||
										!google.data?.canWrite ||
										props.achievementIds().length === 0
									}
									onClick={generate}
								>
									Generate draft
								</Button>
							</div>
						</>
					}
				>
					<Show when={draft.data?.status === "ready"}>
						<p class="mb-3 text-sm text-foreground">Your draft is ready.</p>
						<div class="flex flex-wrap gap-2">
							<Link
								to="/tailoring/drafts/$id"
								params={{ id: draftId() ?? "" }}
								class="inline-flex h-8 items-center rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition hover:bg-primary-hover"
							>
								Review draft
							</Link>
							<a
								href={draft.data?.draftDocUrl ?? undefined}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex h-8 items-center rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
							>
								Open in Google Docs
							</a>
						</div>
					</Show>
					<Show when={draft.data?.status === "failed"}>
						<p class="mb-3 text-sm text-danger">
							Generating the draft failed
							{draft.data?.lastError ? `: ${draft.data.lastError}` : "."}
						</p>
						<Button variant="outline" onClick={retry}>
							Try again
						</Button>
					</Show>
					<Show when={!draft.data || !isSettled(draft.data.status)}>
						<p class="text-sm text-muted" aria-live="polite">
							Writing your draft. This usually takes under a minute.
						</p>
					</Show>
				</Show>
			</Card>
		</div>
	);
}
