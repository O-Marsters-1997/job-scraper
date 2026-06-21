import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { ConflictError } from "../../../api/sourceTargets";
import {
	useCreateSourceTarget,
	useDeleteSourceTarget,
	useSourceTargets,
	useUpdateSourceTarget,
} from "../../../hooks/useSourceTargets";
import type { SourceTarget } from "../../../types/sourceTarget";

export const Route = createFileRoute("/_auth/settings/searches")({
	component: SearchesPage,
});

function SearchesPage() {
	const query = useSourceTargets();
	const createMutation = useCreateSourceTarget();
	const updateMutation = useUpdateSourceTarget();
	const deleteMutation = useDeleteSourceTarget();

	const [showAdd, setShowAdd] = createSignal(false);
	const [newKeywords, setNewKeywords] = createSignal("");
	const [newRegion, setNewRegion] = createSignal("");
	const [newScrapeNow, setNewScrapeNow] = createSignal(false);

	const [conflictError, setConflictError] = createSignal<string | null>(null);
	const [scrapeQueued, setScrapeQueued] = createSignal(false);

	const handleAdd = async () => {
		if (!newKeywords().trim()) return;
		setConflictError(null);
		setScrapeQueued(false);
		try {
			const filters: Record<string, string> = {};
			if (newRegion().trim()) filters.region = newRegion().trim();
			await createMutation.mutateAsync({
				source: "wis",
				value: newKeywords().trim(),
				filters,
				scrape_now: newScrapeNow(),
			});
			if (newScrapeNow()) setScrapeQueued(true);
			setNewKeywords("");
			setNewRegion("");
			setNewScrapeNow(false);
			setShowAdd(false);
		} catch (err) {
			if (err instanceof ConflictError) {
				setConflictError("A search with these keywords and region already exists.");
			} else {
				setConflictError("Failed to add search. Please try again.");
			}
		}
	};

	const handleToggle = (t: SourceTarget) => {
		updateMutation.mutate({ id: t.ID, enabled: !t.Enabled });
	};

	const handleDelete = (id: string) => {
		deleteMutation.mutate(id);
	};

	// Wis searches only for this page
	const searches = () => (query.data ?? []).filter((t) => t.Source === "wis");

	return (
		<div class="max-w-2xl px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Tracked searches
				</h1>
				<p class="mt-0.5 text-xs text-faint">
					Keywords and region filters for workinstartups.com — each search runs
					on its own 6-hour cycle
				</p>
			</div>

			<Show when={scrapeQueued()}>
				<div class="mb-4 rounded-lg border border-primary/30 bg-accent-subtle px-4 py-3 text-sm text-primary">
					Scrape queued — matching jobs will appear shortly.
				</div>
			</Show>

			<Show when={conflictError()}>
				<div class="mb-4 rounded-lg border border-destructive/30 bg-destructive-subtle px-4 py-3 text-sm text-destructive-strong">
					{conflictError()}
				</div>
			</Show>

			<Show when={query.isPending}>
				<p class="text-sm text-muted">Loading…</p>
			</Show>

			<Show when={query.isSuccess}>
				<div class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface">
					<For each={searches()}>
						{(t) => (
							<div class="flex items-center gap-3 px-4 py-3">
								<div class="flex-1 min-w-0">
									<p class="text-sm font-medium text-foreground truncate">
										{t.Value}
									</p>
									<p class="text-xs text-faint">
										{t.Filters.region ? `Region: ${t.Filters.region}` : "Any region"}
									</p>
								</div>
								{/* Toggle */}
								<button
									type="button"
									role="switch"
									aria-checked={t.Enabled}
									onClick={() => handleToggle(t)}
									disabled={updateMutation.isPending}
									title={t.Enabled ? "Disable" : "Enable"}
									class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors focus-visible:outline-none disabled:opacity-50"
									style={{
										"background-color": t.Enabled
											? "var(--color-primary)"
											: "var(--color-border)",
									}}
								>
									<span
										class="inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow transition-transform"
										style={{
											transform: t.Enabled ? "translateX(18px)" : "translateX(2px)",
										}}
									/>
								</button>
								<button
									type="button"
									onClick={() => handleDelete(t.ID)}
									disabled={deleteMutation.isPending}
									class="rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle disabled:opacity-50"
								>
									Delete
								</button>
							</div>
						)}
					</For>

					<Show when={showAdd()}>
						<div class="flex flex-col gap-3 px-4 py-4">
							<div class="flex flex-col gap-2">
								<label class="text-xs font-medium text-foreground">
									Keywords
								</label>
								<input
									class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
									placeholder="e.g. product engineer"
									value={newKeywords()}
									onInput={(e) => setNewKeywords(e.currentTarget.value)}
									onKeyDown={(e) => e.key === "Enter" && handleAdd()}
									autofocus
								/>
							</div>
							<div class="flex flex-col gap-2">
								<label class="text-xs font-medium text-foreground">
									Region{" "}
									<span class="font-normal text-faint">(optional)</span>
								</label>
								<input
									class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
									placeholder="e.g. uk, us, remote"
									value={newRegion()}
									onInput={(e) => setNewRegion(e.currentTarget.value)}
									onKeyDown={(e) => e.key === "Enter" && handleAdd()}
								/>
							</div>
							<label class="flex items-center gap-2 text-xs font-medium text-foreground cursor-pointer">
								<input
									type="checkbox"
									checked={newScrapeNow()}
									onChange={(e) => setNewScrapeNow(e.currentTarget.checked)}
									class="h-4 w-4 rounded border-border accent-primary"
								/>
								Scrape now — get results immediately instead of waiting up to 6 hours
							</label>
							<div class="flex items-center gap-2 pt-1">
								<button
									type="button"
									onClick={handleAdd}
									disabled={createMutation.isPending || !newKeywords().trim()}
									class="rounded px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
								>
									{createMutation.isPending ? "Adding…" : "Add"}
								</button>
								<button
									type="button"
									onClick={() => {
										setShowAdd(false);
										setNewKeywords("");
										setNewRegion("");
										setNewScrapeNow(false);
										setConflictError(null);
									}}
									class="rounded px-3 py-1.5 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
								>
									Cancel
								</button>
							</div>
						</div>
					</Show>
				</div>

				<Show when={!showAdd()}>
					<button
						type="button"
						onClick={() => {
							setScrapeQueued(false);
							setConflictError(null);
							setShowAdd(true);
						}}
						class="mt-4 inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover"
					>
						<svg
							aria-hidden="true"
							width="12"
							height="12"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2.5"
							stroke-linecap="round"
						>
							<line x1="12" y1="5" x2="12" y2="19" />
							<line x1="5" y1="12" x2="19" y2="12" />
						</svg>
						Add search
					</button>
				</Show>

				<p class="mt-3 text-xs text-faint">
					Each search runs independently on the 6-hour cron. Toggle to pause
					without deleting.
				</p>
			</Show>
		</div>
	);
}
