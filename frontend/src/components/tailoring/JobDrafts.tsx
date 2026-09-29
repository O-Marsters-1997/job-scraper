import { Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatDate } from "@/lib/datetime";
import type { Draft } from "@/types/tailoring";
import { useJobDrafts } from "../../hooks/useTailoring";

function draftLabel(d: Draft): string {
	if (d.outcome) return d.outcome === "kept" ? "Kept" : "Discarded";
	switch (d.status) {
		case "ready":
			return "Ready to review";
		case "failed":
			return "Failed";
		default:
			return "Writing";
	}
}

export function JobDrafts(props: { jobId: string }) {
	const drafts = useJobDrafts(() => props.jobId);
	return (
		<Show when={(drafts.data?.length ?? 0) > 0}>
			<Card>
				<CardHeader class="pb-2">
					<CardTitle>Draft CVs</CardTitle>
				</CardHeader>
				<CardContent class="gap-1.5">
					<For each={drafts.data}>
						{(d) => (
							<Link
								to="/tailoring/drafts/$id"
								params={{ id: d.id }}
								class="flex items-center justify-between gap-3 rounded-md px-2 py-1.5 text-xs transition-colors hover:bg-surface-muted"
							>
								<span class="font-mono tabular-nums text-faint">
									{formatDate(d.createdAt)}
								</span>
								<span class="text-foreground">{draftLabel(d)}</span>
							</Link>
						)}
					</For>
				</CardContent>
			</Card>
		</Show>
	);
}
