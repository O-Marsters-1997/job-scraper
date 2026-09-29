import { Link } from "@tanstack/solid-router";
import { For } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

const panel = "rounded-xl border border-border bg-surface p-10 text-center";

export function CVTemplatesLoading() {
	return (
		<Card class="overflow-hidden divide-y divide-border">
			<For each={[1, 2, 3]}>
				{() => (
					<div class="flex items-center gap-4 px-4 py-3">
						<div class="h-4 w-48 animate-pulse rounded bg-surface-muted" />
						<div class="h-4 w-32 animate-pulse rounded bg-surface-muted" />
						<div class="ml-auto h-4 w-24 animate-pulse rounded bg-surface-muted" />
					</div>
				)}
			</For>
		</Card>
	);
}

export function CVTemplatesNotConnected() {
	return (
		<div class={panel}>
			<p class="text-sm text-muted">
				Connect your Google account in{" "}
				<Link
					to="/settings/integrations"
					class="font-medium text-primary underline-offset-2 hover:underline"
				>
					Settings → Integrations
				</Link>{" "}
				to see your CVs here.
			</p>
		</div>
	);
}

export function CVTemplatesError(props: { message: string | undefined }) {
	return (
		<div class="rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
			<p class="mb-1 text-sm font-semibold text-destructive-strong">Error</p>
			<p class="text-sm text-muted">{props.message}</p>
		</div>
	);
}

export function CVTemplatesEmpty(props: { onAdd: () => void }) {
	return (
		<div class={panel}>
			<p class="mb-3 text-sm text-muted">
				Add a Google Doc to see your CVs here.
			</p>
			<Button onClick={() => props.onAdd()}>
				<Icon name="plus" size={12} strokeWidth={2.5} />
				Add doc
			</Button>
		</div>
	);
}

export function CVTemplatesAllHidden(props: { onShowHidden: () => void }) {
	return (
		<div class={panel}>
			<p class="mb-1 text-sm text-muted">All CVs are hidden.</p>
			<p class="text-xs text-faint">
				Toggle{" "}
				<button
					type="button"
					class="font-medium text-primary underline-offset-2 hover:underline"
					onClick={() => props.onShowHidden()}
				>
					Show hidden
				</button>{" "}
				to restore them.
			</p>
		</div>
	);
}
