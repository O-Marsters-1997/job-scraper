import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { AddDocDialog } from "@/components/cv/AddDocDialog";
import { PageHeading } from "@/components/PageHeading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	cvTemplatesQueryOptions,
	useCVTemplates,
	useHideTab,
	useShowTab,
} from "../../hooks/useCVTemplates";
import { useTableSort } from "../../hooks/useTableSort";
import { formatDate } from "../../lib/datetime";
import { queryClient } from "../../lib/queryClient";
import type { CV } from "../../types/cv";

export const Route = createFileRoute("/_auth/cv-templates")({
	loader: () => queryClient.ensureQueryData(cvTemplatesQueryOptions),
	component: CVTemplatesPage,
});

type SortKey = "Title" | "ModifiedAt";

function CVTemplatesPage() {
	const query = useCVTemplates();
	const navigate = useNavigate();
	const hideMutation = useHideTab();
	const showMutation = useShowTab();

	const handleHide = (cv: CV) => {
		hideMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const handleRestore = (cv: CV) => {
		showMutation.mutate({ docId: cv.DocID, tabId: cv.TabID });
	};

	const [searchQuery, setSearchQuery] = createSignal("");
	const [showHidden, setShowHidden] = createSignal(false);
	const [dialogOpen, setDialogOpen] = createSignal(false);

	const { sortKey, sortDir, handleSort, sortIcon } =
		useTableSort<SortKey>("Title");

	const thClass = (key: SortKey) =>
		`select-none transition-colors hover:text-foreground${sortKey() === key ? " text-foreground" : ""}`;

	const ariaSort = (key: SortKey): "ascending" | "descending" | "none" => {
		if (sortKey() !== key) return "none";
		return sortDir() === "asc" ? "ascending" : "descending";
	};

	const filteredSorted = createMemo<CV[]>(() => {
		const q = searchQuery().toLowerCase();
		const data = query.data ?? [];
		const visibilityFiltered = showHidden()
			? data
			: data.filter((cv) => cv.Visible);
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

	const isNotConnected = () =>
		!!query.error?.message?.toLowerCase().includes("not connected");

	return (
		<div class="px-7 py-6">
			<PageHeading title="CVs" subtitle="Google Docs tracked as CVs">
				<Button class="shrink-0" onClick={() => setDialogOpen(true)}>
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
				</Button>
			</PageHeading>

			<Show when={query.isPending}>
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
						<Button onClick={() => setDialogOpen(true)}>
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
						</Button>
					</div>
				</Show>

				<Show when={(query.data?.length ?? 0) > 0}>
					<div class="mb-3 flex items-center gap-3">
						<Label for="cv-search" class="sr-only">
							Search CVs
						</Label>
						<Input
							id="cv-search"
							type="search"
							placeholder="Search by title or source…"
							value={searchQuery()}
							onInput={(e) => setSearchQuery(e.currentTarget.value)}
							class="max-w-xs"
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
						when={
							filteredSorted().length === 0 &&
							!searchQuery() &&
							!showHidden() &&
							hasHiddenCVs()
						}
					>
						<div class="rounded-xl border border-border bg-surface p-10 text-center">
							<p class="mb-1 text-sm text-muted">All CVs are hidden.</p>
							<p class="text-xs text-faint">
								Toggle{" "}
								<button
									type="button"
									class="font-medium text-primary underline-offset-2 hover:underline"
									onClick={() => setShowHidden(true)}
								>
									Show hidden
								</button>{" "}
								to restore them.
							</p>
						</div>
					</Show>

					<Show when={filteredSorted().length > 0 || searchQuery()}>
						<Card class="overflow-hidden">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead aria-sort={ariaSort("Title")}>
											<button
												type="button"
												onClick={() => handleSort("Title")}
												class={thClass("Title")}
											>
												Title{" "}
												<span class="font-mono text-2xs" aria-hidden="true">
													{sortIcon("Title")}
												</span>
											</button>
										</TableHead>
										<TableHead>Source doc</TableHead>
										<TableHead aria-sort={ariaSort("ModifiedAt")}>
											<button
												type="button"
												onClick={() => handleSort("ModifiedAt")}
												class={thClass("ModifiedAt")}
											>
												Last modified{" "}
												<span class="font-mono text-2xs" aria-hidden="true">
													{sortIcon("ModifiedAt")}
												</span>
											</button>
										</TableHead>
										<TableHead class="w-10" />
									</TableRow>
								</TableHeader>
								<TableBody>
									<For each={filteredSorted()}>
										{(cv) => (
											<TableRow
												class={`cursor-pointer${!cv.Visible ? " opacity-60" : ""}`}
												onClick={() => {
													navigate({
														to: "/cv-templates/$docId/$tabId",
														params: { docId: cv.DocID, tabId: cv.TabID },
													});
												}}
											>
												<TableCell class="py-2.5">
													<div class="flex items-center gap-2">
														<Link
															to="/cv-templates/$docId/$tabId"
															params={{ docId: cv.DocID, tabId: cv.TabID }}
															onClick={(e) => e.stopPropagation()}
															class="font-medium text-foreground underline-offset-2 hover:text-primary hover:underline"
														>
															{cv.Title || "—"}
														</Link>
														<a
															href={cv.DocURL}
															target="_blank"
															rel="noreferrer"
															onClick={(e) => e.stopPropagation()}
															title="Open in Google Docs"
															class="text-faint transition-colors hover:text-foreground"
														>
															<svg
																aria-hidden="true"
																width="12"
																height="12"
																viewBox="0 0 24 24"
																fill="none"
																stroke="currentColor"
																stroke-width="2"
																stroke-linecap="round"
																stroke-linejoin="round"
															>
																<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
																<polyline points="15 3 21 3 21 9" />
																<line x1="10" y1="14" x2="21" y2="3" />
															</svg>
															<span class="sr-only">
																Open {cv.Title || "document"} in Google Docs
															</span>
														</a>
														<Show when={!cv.Visible}>
															<Badge variant="secondary">Hidden</Badge>
														</Show>
													</div>
												</TableCell>
												<TableCell class="py-2.5 text-muted">
													{cv.SourceDoc || "—"}
												</TableCell>
												<TableCell class="py-2.5">
													<span class="font-mono text-xs tabular-nums text-faint">
														{formatDate(cv.ModifiedAt)}
													</span>
												</TableCell>
												<TableCell
													class="py-2.5 text-right"
													onClick={(e) => e.stopPropagation()}
													onKeyDown={(e) => e.stopPropagation()}
												>
													<Show when={cv.Visible}>
														<button
															type="button"
															title="Hide tab"
															onClick={() => handleHide(cv)}
															disabled={hideMutation.isPending}
															class="inline-flex h-7 w-7 items-center justify-center rounded-md text-faint transition-colors hover:bg-destructive-subtle hover:text-destructive-strong disabled:opacity-50"
														>
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
												</TableCell>
											</TableRow>
										)}
									</For>
								</TableBody>
							</Table>

							<Show when={filteredSorted().length === 0 && searchQuery()}>
								<div class="px-4 py-8 text-center text-sm text-muted">
									No CVs match "{searchQuery()}"
								</div>
							</Show>
						</Card>
					</Show>
				</Show>
			</Show>

			<AddDocDialog open={dialogOpen()} onOpenChange={setDialogOpen} />
		</div>
	);
}
