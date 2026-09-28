import { createFileRoute } from "@tanstack/solid-router";
import { Show } from "solid-js";
import { ErrorState } from "@/components/ErrorState";
import { Icon } from "@/components/Icon";
import { API_BASE } from "../../../api/config";
import { useDisconnectGoogle, useGoogleStatus } from "../../../hooks/useGoogle";

export const Route = createFileRoute("/_auth/settings/integrations")({
	component: IntegrationsPage,
});

function IntegrationsPage() {
	const status = useGoogleStatus();
	const disconnect = useDisconnectGoogle();

	return (
		<>
			<div>
				<div class="flex items-center gap-4">
					<div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-border bg-surface-muted">
						<Icon name="file" size={18} />
					</div>

					<div class="flex-1 min-w-0">
						<p class="text-sm font-medium text-foreground">Google Account</p>
						<Show
							when={!status.isPending}
							fallback={
								<div class="mt-1 h-3.5 w-40 rounded bg-surface-muted" />
							}
						>
							<Show
								when={status.data?.connected}
								fallback={
									<p class="mt-0.5 text-xs text-faint">Not connected</p>
								}
							>
								<p class="mt-0.5 text-xs text-faint">
									Connected as{" "}
									<span class="text-muted">{status.data?.email}</span>
								</p>
							</Show>
						</Show>
					</div>

					<Show
						when={!status.isPending}
						fallback={<div class="h-8 w-24 rounded-md bg-surface-muted" />}
					>
						<Show
							when={status.data?.connected}
							fallback={
								<a
									href={`${API_BASE}/google/oauth/start`}
									class="inline-flex h-8 items-center rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition hover:bg-primary-hover"
								>
									Connect Google
								</a>
							}
						>
							<button
								type="button"
								onClick={() => disconnect.mutate()}
								disabled={disconnect.isPending}
								class="inline-flex h-8 items-center rounded-md border border-border bg-surface px-3 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle hover:border-destructive/30 disabled:opacity-50"
							>
								Disconnect
							</button>
						</Show>
					</Show>
				</div>
				<p class="mt-3 text-xs text-faint">
					Google Drive access lets FastTrack read your CV documents.
				</p>
			</div>

			<Show when={status.isError}>
				<ErrorState
					error={status.error}
					onRetry={() => status.refetch()}
					message="Could not load Google integration status."
				/>
			</Show>
		</>
	);
}
