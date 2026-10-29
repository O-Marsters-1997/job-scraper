import { createSignal, For, Show } from "solid-js";
import { useExperience } from "@/hooks/useExperience";
import { useSaveHeadings } from "@/hooks/useTailoring";
import { loadedData } from "@/lib/queryData";
import type { Draft } from "@/types/tailoring";

export function UnlinkedRoles(props: { draft: Draft }) {
	const positions = useExperience();
	const save = useSaveHeadings();
	const [linked, setLinked] = createSignal<string[]>([]);
	const unlinked = () =>
		(props.draft.content?.positions ?? []).filter(
			(p) => !p.linked && !linked().includes(p.headingText),
		);

	const link = (headingText: string, positionId: string) =>
		save.mutate(
			{
				docId: props.draft.baseDocId,
				tabId: props.draft.baseTabId,
				mappings: [{ headingText, positionId }],
			},
			{ onSuccess: () => setLinked((l) => [...l, headingText]) },
		);

	return (
		<Show when={unlinked().length > 0}>
			<div class="border-b border-border bg-surface px-4 py-3 text-sm">
				<p class="mb-2 text-foreground">
					These roles are not linked to your Experience Bank. Link one so the
					next Draft from this CV can tailor it.
				</p>
				<For each={unlinked()}>
					{(p) => (
						<label class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,18rem)] items-center gap-3 py-1">
							<span class="truncate text-muted">{p.headingText}</span>
							<span class="rounded bg-border/60 px-1.5 py-0.5 text-xs text-muted">
								Not linked to bank
							</span>
							<select
								aria-label={`Link ${p.headingText} to a Position`}
								class="h-8 rounded-md border border-border bg-surface px-2 text-sm text-foreground"
								disabled={save.isPending}
								onChange={(e) => link(p.headingText, e.currentTarget.value)}
							>
								<option value="" disabled selected>
									Choose a Position
								</option>
								<For each={loadedData(positions) ?? []}>
									{(pos) => (
										<option value={pos.id}>
											{pos.title}, {pos.employer}
										</option>
									)}
								</For>
							</select>
						</label>
					)}
				</For>
				<Show when={save.error}>
					<p class="mt-2 text-destructive-strong">Could not save the link.</p>
				</Show>
			</div>
		</Show>
	);
}
