import { Link } from "@tanstack/solid-router";
import { type JSX, Show } from "solid-js";
import { googleWriteHref } from "@/components/GoogleWriteConsent";
import { Button } from "@/components/ui/button";
import type { Draft } from "@/types/tailoring";
import { useGoogleStatus } from "../../hooks/useGoogle";
import type { KeepFlow } from "../../hooks/useKeepFlow";

export function KeepControls(props: { draft: Draft; flow: KeepFlow }) {
	return (
		<Show
			when={props.draft.status === "keeping"}
			fallback={
				<Show
					when={props.draft.outcome === null}
					fallback={
						<span class="px-2 text-sm text-muted">
							{props.draft.outcome === "kept" ? "Kept" : "Discarded"}
						</span>
					}
				>
					<Button
						variant="outline"
						size="sm"
						disabled={props.flow.discardMutation.isPending}
						onClick={() => props.flow.discard()}
					>
						Discard
					</Button>
					<Button
						size="sm"
						disabled={props.flow.keepMutation.isPending}
						onClick={() => props.flow.keep()}
					>
						Keep draft
					</Button>
				</Show>
			}
		>
			<span class="px-2 text-sm text-muted" aria-live="polite">
				Keeping…
			</span>
		</Show>
	);
}

function Notice(props: { children: JSX.Element; alert?: boolean }) {
	return (
		<div
			role={props.alert ? "alert" : undefined}
			class="border-b border-border bg-surface px-4 py-3 text-sm"
		>
			{props.children}
		</div>
	);
}

function ErrorNotice(props: { children: JSX.Element }) {
	return (
		<Notice alert>
			<p class="text-destructive-strong">{props.children}</p>
		</Notice>
	);
}

export function KeepNotices(props: {
	draft: Draft;
	editable: boolean;
	flow: KeepFlow;
}) {
	const google = useGoogleStatus();
	return (
		<>
			<Show
				when={
					props.editable && google.data?.connected && !google.data.canEditDocs
				}
			>
				<Notice>
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
				</Notice>
			</Show>
			<Show when={props.flow.blockedByKept()}>
				<Notice alert>
					<p class="mb-3 text-foreground">
						This job already has a kept draft. A job keeps one draft at a time.
					</p>
					<div class="flex flex-wrap gap-2">
						<Show when={props.flow.otherKept()?.id}>
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
							disabled={
								!props.flow.otherKept() ||
								props.flow.discardMutation.isPending ||
								props.flow.keepMutation.isPending
							}
							onClick={() => props.flow.replaceKept()}
						>
							Discard the kept draft and keep this one
						</Button>
					</div>
				</Notice>
			</Show>
			<Show
				when={props.flow.keepMutation.isError && !props.flow.blockedByKept()}
			>
				<ErrorNotice>Could not keep the draft. Try again.</ErrorNotice>
			</Show>
			<Show when={props.editable && props.draft.lastError}>
				<ErrorNotice>
					Keeping the draft failed: {props.draft.lastError}
				</ErrorNotice>
			</Show>
			<Show when={props.flow.discardMutation.isError}>
				<ErrorNotice>Could not discard the draft. Try again.</ErrorNotice>
			</Show>
		</>
	);
}
