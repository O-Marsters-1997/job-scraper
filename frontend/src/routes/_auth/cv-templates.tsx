import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { batch, createMemo, createSignal, For, Show } from "solid-js";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import {
	cvTemplatesQueryOptions,
	useAddTrackedDoc,
	useCVTemplates,
	useHideTab,
	useShowTab,
} from "../../hooks/useCVTemplates";
import { queryClient } from "../../lib/queryClient";
import type { CV } from "../../types/cv";

export const Route = createFileRoute("/_auth/cv-templates")({
	loader: () => queryClient.ensureQueryData(cvTemplatesQueryOptions),
	component: CVTemplatesPage,
});

type SortKey = "Title" | "ModifiedAt";
type SortDir = "asc" | "desc";

function formatDate(iso: string): string {
	if (!iso) return "—";
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return "—";
	return d.toLocaleDateString("en-GB", {
		day: "numeric",
		month: "short",
		year: "numeric",
	});
}

function CVTemplatesPage() {
	const query = useCVTemplates();
	const navigate = useNavigate();
	const addMutation = useAddTrackedDoc();
	const hideMutation = useHideTab();
	const showMutation = useShowTab();

	const handleHide = (cv: CV) => {
		hideMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const handleRestore = (cv: CV) => {
		showMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const [searchQuery, setSearchQuery] = createSignal("");
	const [sortKey, setSortKey] = createSignal<SortKey>("Title");
	const [sortDir, setSortDir] = createSignal<SortDir>("asc");
	const [showHidden, setShowHidden] = createSignal(false);

	const [dialogOpen, setDialogOpen] = createSignal(false);
	const [docUrl, setDocUrl] = createSignal("");
	const [urlError, setUrlError] = createSignal<string | null>(null);

	const handleSort = (key: SortKey) => {
		if (sortKey() === key) {
			setSortDir((d) => (d === "asc" ? "desc" : "asc"));
		} else {
			batch(() => {
				setSortKey(key);
				setSortDir("asc");
			});
		}
	};

	const sortIcon = (key: SortKey) => {
		if (sortKey() !== key) return null;
		return sortDir() === "asc" ? "↑" : "↓";
	};

	const filteredSorted = createMemo<CV[]>(() => {
		const q = searchQuery().toLowerCase();
		const data = query.data ?? [];
		const visibilityFiltered = showHidden() ? data : data.filter((cv) => cv.Visible);
		const filtered = q
			? visibilityFiltered.filter(
					(cv) =>
						cv.Title.toLowerCase().includes(q) ||
						cv.SourceDoc.toLowerCase().includes(q),
				)
			: visibilityFiltered;

		const key = sortKey();
		const dir = sortDir();
		return [...filtered].sort((a, b) => {
			const av = a[key];
			const bv = b[key];
			const cmp = av < bv ? -1 : av > bv ? 1 : 0;
			return dir === "asc" ? cmp : -cmp;
		});
	});

	const hasHiddenCVs = () => (query.data ?? []).some((cv) => !cv.Visible);

	const openDialog = () => {
		batch(() => {
			setDocUrl("");
			setUrlError(null);
			setDialogOpen(true);
		});
	};

	const handleAdd = async () => {
		setUrlError(null);
		try {
			await addMutation.mutateAsync(docUrl().trim());
			setDialogOpen(false);
		} catch (err) {
			if (err instanceof Error && err.message === "invalid-url") {
				setUrlError("Invalid Google Docs URL or ID");
			} else if (err instanceof Error && err.message === "access-denied") {
				setUrlError(
					"Cannot access this document. Make sure it's shared with your Google account.",
				);
			} else {
				setUrlError("Something went wrong. Please try again.");
			}
		}
	};

	const isNotConnected = () =>
		!!query.error?.message?.toLowerCase().includes("not connected");

	const thClass = (key: SortKey) =>
		`h-9 cursor-pointer select-none px-4 text-left text-xs font-semibold uppercase tracking-wide text-faint transition-colors hover:text-foreground ${sortKey() === key ? "text-foreground" : ""}`;

	return (
		<div class="px-7 py-6">
			<div class="mb-5 flex items-start justify-between gap-4">
				<div>
					<h1 class="text-lg font-bold tracking-tight text-foreground">
						CVs
					</h1>
					<p class="mt-0.5 text-xs text-faint">
						Google Docs tracked as CVs
					</p>
				</div>
				<button
					type="button"
					onClick={openDialog}
					class="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary-hover"
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
					Add doc
				</button>
			</div>

			<Show when={query.isPending}>
				<div class="divide-y divide-border overflow-hidden rounded-xl border border-border bg-surface">
					<For each={[1, 2, 3]}>
						{() => (
							<div class="flex items-center gap-4 px-4 py-3">
								<div class="h-4 w-48 animate-pulse rounded bg-surface-muted" />
								<div class="h-4 w-32 animate-pulse rounded bg-surface-muted" />
								<div class="ml-auto h-4 w-24 animate-pulse rounded bg-surface-muted" />
							</div>
						)}
					</For>
				</div>
			</Show>

			<Show when={query.isError && isNotConnected()}>
				<div class="rounded-xl border border-border bg-surface p-10 text-center">
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
			</Show>

			<Show when={query.isError && !isNotConnected()}>
				<div class="rounded-xl border border-destructive/30 bg-destructive-subtle p-6">
					<p class="mb-1 text-sm font-semibold text-destructive-strong">
						Error
					</p>
					<p class="text-sm text-muted">{query.error?.message}</p>
				</div>
			</Show>

			<Show when={query.isSuccess}>
				<Show when={(query.data?.length ?? 0) === 0}>
					<div class="rounded-xl border border-border bg-surface p-10 text-center">
						<p class="mb-3 text-sm text-muted">
							Add a Google Doc to see your CVs here.
						</p>
						<button
							type="button"
							onClick={openDialog}
							class="inline-flex h-9 items-center gap-1.5 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary-hover"
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
							Add doc
						</button>
					</div>
				</Show>

				<Show when={(query.data?.length ?? 0) > 0}>
					<div class="mb-3 flex items-center gap-3">
						<input
							type="search"
							placeholder="Search by title or source…"
							value={searchQuery()}
							onInput={(e) => setSearchQuery(e.currentTarget.value)}
							class="h-9 w-full max-w-xs rounded-md border border-border bg-surface px-3 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/10"
						/>
						<Switch checked={showHidden()} onChange={setShowHidden}>
							<SwitchLabel class="inline-flex items-center gap-2">
								<SwitchControl>
									<SwitchThumb />
								</SwitchControl>
								Show hidden
							</SwitchLabel>
						</Switch>
					</div>

					<Show
						when={filteredSorted().length === 0 && !searchQuery() && !showHidden() && hasHiddenCVs()}
					>
						<div class="rounded-xl border border-border bg-surface p-10 text-center">
							<p class="mb-1 text-sm text-muted">All CVs are hidden.</p>
							<p class="text-xs text-faint">
								Toggle{" "}
								<button
									type="button"
									class="font-medium text-primary hover:underline underline-offset-2"
									onClick={() => setShowHidden(true)}
								>
									Show hidden
								</button>{" "}
								to restore them.
							</p>
						</div>
					</Show>

					<Show when={filteredSorted().length > 0 || searchQuery()}>
						<div class="overflow-hidden rounded-xl border border-border bg-surface">
							<div class="overflow-x-auto">
								<table class="w-full text-sm">
									<thead class="border-b border-border bg-surface-muted">
										<tr>
											<th
												class={thClass("Title")}
												onClick={() => handleSort("Title")}
											>
												Title{" "}
												<span class="font-mono text-[10px]">
													{sortIcon("Title")}
												</span>
											</th>
											<th class="h-9 px-4 text-left text-xs font-semibold uppercase tracking-wide text-faint">
												Source doc
											</th>
											<th
												class={thClass("ModifiedAt")}
												onClick={() => handleSort("ModifiedAt")}
											>
												Last modified{" "}
												<span class="font-mono text-[10px]">
													{sortIcon("ModifiedAt")}
												</span>
											</th>
											<th class="h-9 w-10 px-4" />
										</tr>
									</thead>
									<tbody class="divide-y divide-border">
										<For each={filteredSorted()}>
											{(cv) => (
												<tr
													class={`cursor-pointer transition-colors hover:bg-surface-muted ${!cv.Visible ? "opacity-60" : ""}`}
													onClick={() => {
														navigate({
															to: "/cv-templates/$docId/$tabId",
															params: { docId: cv.DocID, tabId: cv.TabID },
														});
													}}
												>
													<td class="px-4 py-2.5">
														<div class="flex items-center gap-2">
															<a
																href={cv.DocURL}
																target="_blank"
																rel="noreferrer"
																onClick={(e) => e.stopPropagation()}
																class="font-medium text-foreground hover:text-primary hover:underline underline-offset-2"
															>
																{cv.Title || "—"}
															</a>
															<Show when={!cv.Visible}>
																<Badge variant="secondary">Hidden</Badge>
															</Show>
														</div>
													</td>
													<td class="px-4 py-2.5 text-muted">
														{cv.SourceDoc || "—"}
													</td>
													<td class="px-4 py-2.5">
														<span class="font-mono text-xs tabular-nums text-faint">
															{formatDate(cv.ModifiedAt)}
														</span>
													</td>
													<td
														class="px-4 py-2.5 text-right"
														onClick={(e) => e.stopPropagation()}
													>
														<Show when={cv.Visible}>
															<button
																type="button"
																title="Hide tab"
																onClick={() => handleHide(cv)}
																disabled={hideMutation.isPending}
																class="inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive disabled:opacity-50"
															>
																{/* trash */}
																<svg
																	aria-hidden="true"
																	width="14"
																	height="14"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																>
																	<polyline points="3 6 5 6 21 6" />
																	<path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
																	<path d="M10 11v6" />
																	<path d="M14 11v6" />
																	<path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
																</svg>
															</button>
														</Show>
														<Show when={!cv.Visible}>
															<button
																type="button"
																title="Restore tab"
																onClick={() => handleRestore(cv)}
																disabled={showMutation.isPending}
																class="inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-accent-subtle hover:text-primary disabled:opacity-50"
															>
																{/* rotate-ccw (restore) */}
																<svg
																	aria-hidden="true"
																	width="14"
																	height="14"
																	viewBox="0 0 24 24"
																	fill="none"
																	stroke="currentColor"
																	stroke-width="2"
																	stroke-linecap="round"
																	stroke-linejoin="round"
																>
																	<path d="M1 4v6h6" />
																	<path d="M3.51 15a9 9 0 1 0 .49-3.5" />
																</svg>
															</button>
														</Show>
													</td>
												</tr>
											)}
										</For>
									</tbody>
								</table>
							</div>

							<Show when={filteredSorted().length === 0 && searchQuery()}>
								<div class="px-4 py-8 text-center text-sm text-muted">
									No CVs match "{searchQuery()}"
								</div>
							</Show>
						</div>
					</Show>
				</Show>
			</Show>

			<Dialog open={dialogOpen()} onOpenChange={setDialogOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Add Google Doc</DialogTitle>
						<p class="text-sm text-faint">
							Paste the URL of a Google Doc to track it as a CV template.
						</p>
					</DialogHeader>

					<div class="flex flex-col gap-3">
						<div>
							<Label>Google Docs URL</Label>
							<Input
								type="url"
								placeholder="https://docs.google.com/document/d/…"
								value={docUrl()}
								onInput={(e) => {
									setDocUrl(e.currentTarget.value);
									setUrlError(null);
								}}
								onKeyDown={(e) => e.key === "Enter" && handleAdd()}
							/>
							<Show when={urlError()}>
								<p class="mt-1.5 text-xs text-destructive">{urlError()}</p>
							</Show>
						</div>
					</div>

					<DialogFooter class="border-t border-border pt-4">
						<Button
							variant="outline"
							onClick={() => setDialogOpen(false)}
						>
							Cancel
						</Button>
						<Button
							onClick={handleAdd}
							disabled={addMutation.isPending || !docUrl().trim()}
						>
							Add doc
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
