import { createQuery } from "@tanstack/solid-query";
import { createFileRoute } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { QuotaBar } from "@/components/QuotaBar";
import { SkeletonList } from "@/components/ui/skeleton";
import { meQueryOptions } from "../../../hooks/useAuth";
import { useIsAdmin } from "../../../hooks/useIsAdmin";
import { useProxyUsage } from "../../../hooks/useProxyUsage";
import { formatRelative } from "../../../lib/datetime";
import type { Quota } from "../../../types/quota";

export const Route = createFileRoute("/_auth/settings/usage")({
	component: UsagePage,
});

const PROVIDER_LABELS: Record<string, { name: string; envVar: string }> = {
	decodo: { name: "Decodo", envVar: "DECODO_API_KEY" },
};

function UsagePage() {
	const me = createQuery(() => meQueryOptions);
	const isAdmin = useIsAdmin();
	const query = useProxyUsage();
	return (
		<Show when={!me.isPending} fallback={<SkeletonList rows={1} />}>
			<Show
				when={isAdmin()}
				fallback={<p class="text-sm text-faint">Admins only.</p>}
			>
				<QueryBoundary query={query} fallbackRows={1}>
					{(data) => (
						<For each={data().providers}>
							{(quota) => <ProviderRow quota={quota} />}
						</For>
					)}
				</QueryBoundary>
			</Show>
		</Show>
	);
}

function footnote(q: Quota): string | null {
	if (q.status === "ok" && q.fetchedAt) {
		return `Updated ${formatRelative(q.fetchedAt)}`;
	}
	if (q.status !== "error") return null;
	return q.fetchedAt
		? `Last fetched ${formatRelative(q.fetchedAt)}: ${q.error}`
		: `Not fetched yet: ${q.error}`;
}

function ProviderRow(props: { quota: Quota }) {
	const label = () =>
		PROVIDER_LABELS[props.quota.provider] ?? {
			name: props.quota.provider,
			envVar: "its API key",
		};
	return (
		<section class="flex flex-col gap-2">
			<h2 class="text-sm font-medium text-foreground">{label().name}</h2>
			<Show
				when={props.quota.status !== "not_configured"}
				fallback={
					<p class="text-xs text-faint">Not configured: set {label().envVar}</p>
				}
			>
				<QuotaBar quota={props.quota} />
				<Show when={footnote(props.quota)}>
					{(text) => <p class="text-xs text-faint">{text()}</p>}
				</Show>
			</Show>
		</section>
	);
}
