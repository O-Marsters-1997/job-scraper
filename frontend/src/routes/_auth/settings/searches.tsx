import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { ConflictError } from "../../../api/sourceTargets";
import {
	useCreateSourceTarget,
	useDeleteSourceTarget,
	useSourceTargets,
	useUpdateSourceTarget,
} from "../../../hooks/useSourceTargets";
import { useSources } from "../../../hooks/useSources";
import type { SourceTarget } from "../../../types/sourceTarget";
import type { SourceInfo } from "../../../types/source";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";

export const Route = createFileRoute("/_auth/settings/searches")({
	component: SearchesPage,
});

function SearchesPage() {
	const query = useSourceTargets();
	const sourcesQuery = useSources();
	const createMutation = useCreateSourceTarget();
	const updateMutation = useUpdateSourceTarget();
	const deleteMutation = useDeleteSourceTarget();

	const [showAdd, setShowAdd] = createSignal(false);
	const [selectedSource, setSelectedSource] = createSignal("wis");
	const [newValue, setNewValue] = createSignal("");
	const [newFilters, setNewFilters] = createSignal<Record<string, string>>({});
	const [newScrapeNow, setNewScrapeNow] = createSignal(false);
	const [conflictError, setConflictError] = createSignal<string | null>(null);
	const [scrapeQueued, setScrapeQueued] = createSignal(false);

	const currentSourceInfo = () =>
		(sourcesQuery.data ?? []).find((s) => s.name === selectedSource());

	const sourceLabel = (name: string) =>
		(sourcesQuery.data ?? []).find((s) => s.name === name)?.label ?? name;

	const filterSummary = (t: SourceTarget) => {
		const entries = Object.entries(t.Filters);
		if (entries.length === 0) return "—";
		const info = (sourcesQuery.data ?? []).find((s) => s.name === t.Source);
		return entries
			.map(([k, v]) => {
				const fieldLabel = info?.filters.find((f) => f.name === k)?.label ?? k;
				return `${fieldLabel}: ${v}`;
			})
			.join(", ");
	};

	const resetForm = () => {
		setNewValue("");
		setNewFilters({});
		setNewScrapeNow(false);
		setConflictError(null);
	};

	const handleAdd = async () => {
		const info = currentSourceInfo();
		const val = newValue().trim();
		if (!val) return;

		if (info?.kind === "url" && !val.startsWith(info.url_prefix)) {
			setConflictError(`URL must start with ${info.url_prefix}`);
			return;
		}

		setConflictError(null);
		setScrapeQueued(false);
		try {
			const filters = info?.kind === "filter" ? newFilters() : {};
			await createMutation.mutateAsync({
				source: selectedSource(),
				value: val,
				filters,
				scrape_now: newScrapeNow(),
			});
			if (newScrapeNow()) setScrapeQueued(true);
			resetForm();
			setShowAdd(false);
		} catch (err) {
			if (err instanceof ConflictError) {
				setConflictError("A search with these settings already exists.");
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

	return (
		<div class="max-w-2xl px-7 py-6">
			<div class="mb-5">
				<h1 class="text-lg font-bold tracking-tight text-foreground">
					Tracked searches
				</h1>
				<p class="mt-0.5 text-xs text-faint">
					Keywords, board tokens and URLs to track across supported job sources
					— each runs on its own 6-hour cycle
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
				<Show when={(query.data ?? []).length > 0}>
					<div class="mb-4 overflow-hidden rounded-xl border border-border bg-surface">
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Source</TableHead>
									<TableHead>Search</TableHead>
									<TableHead>Filters</TableHead>
									<TableHead class="w-16" />
									<TableHead class="w-20" />
								</TableRow>
							</TableHeader>
							<TableBody>
								<For each={query.data}>
									{(t) => (
										<TableRow>
											<TableCell>
												<Badge variant="source">
													{sourceLabel(t.Source)}
												</Badge>
											</TableCell>
											<TableCell class="max-w-[200px] truncate font-mono text-xs">
												{t.Value}
											</TableCell>
											<TableCell class="text-faint text-xs">
												{filterSummary(t)}
											</TableCell>
											<TableCell>
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
															transform: t.Enabled
																? "translateX(18px)"
																: "translateX(2px)",
														}}
													/>
												</button>
											</TableCell>
											<TableCell>
												<button
													type="button"
													onClick={() => handleDelete(t.ID)}
													disabled={deleteMutation.isPending}
													class="rounded px-2 py-1 text-xs font-medium text-destructive transition hover:bg-destructive-subtle disabled:opacity-50"
												>
													Delete
												</button>
											</TableCell>
										</TableRow>
									)}
								</For>
							</TableBody>
						</Table>
					</div>
				</Show>

				<Show when={showAdd()}>
					<div class="mb-4 rounded-xl border border-border bg-surface px-4 py-4">
						<Show
							when={sourcesQuery.isSuccess}
							fallback={
								<p class="text-sm text-muted">Loading sources…</p>
							}
						>
							<div class="flex flex-col gap-3">
								<div class="flex flex-col gap-2">
									<p class="text-xs font-medium text-foreground">Source</p>
									<Select<SourceInfo>
										options={sourcesQuery.data ?? []}
										optionValue="name"
										optionTextValue="label"
										value={currentSourceInfo() ?? null}
										onChange={(s) => {
											if (s) {
												setSelectedSource(s.name);
												setNewValue("");
												setNewFilters({});
											}
										}}
										placeholder="Select a source"
										itemComponent={(props) => (
											<SelectItem item={props.item}>
												<SelectItemLabel>
													{props.item.rawValue.label}
												</SelectItemLabel>
											</SelectItem>
										)}
									>
										<SelectTrigger>
											<Select.Value<SourceInfo>>
												{(state) =>
													state.selectedOption()?.label ?? "Select a source"
												}
											</Select.Value>
										</SelectTrigger>
										<SelectContent />
									</Select>
								</div>

								{/* Filter source (WIS): keywords + filter fields */}
								<Show when={currentSourceInfo()?.kind === "filter"}>
									<div class="flex flex-col gap-2">
										<label
											for="new-keywords"
											class="text-xs font-medium text-foreground"
										>
											Keywords
										</label>
										<input
											id="new-keywords"
											class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
											placeholder="e.g. product engineer"
											value={newValue()}
											onInput={(e) => setNewValue(e.currentTarget.value)}
											onKeyDown={(e) => e.key === "Enter" && handleAdd()}
										/>
									</div>
									<For each={currentSourceInfo()?.filters ?? []}>
										{(field) => (
											<div class="flex flex-col gap-2">
												<label
													for={`filter-${field.name}`}
													class="text-xs font-medium text-foreground"
												>
													{field.label}{" "}
													<Show when={!field.required}>
														<span class="font-normal text-faint">
															(optional)
														</span>
													</Show>
												</label>
												<input
													id={`filter-${field.name}`}
													class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
													placeholder={
														field.name === "region"
															? "e.g. uk, us, remote"
															: ""
													}
													value={newFilters()[field.name] ?? ""}
													onInput={(e) =>
														setNewFilters((prev) => ({
															...prev,
															[field.name]: e.currentTarget.value,
														}))
													}
													onKeyDown={(e) => e.key === "Enter" && handleAdd()}
												/>
											</div>
										)}
									</For>
								</Show>

								{/* Board source: board token */}
								<Show when={currentSourceInfo()?.kind === "board"}>
									<div class="flex flex-col gap-2">
										<label
											for="new-board-token"
											class="text-xs font-medium text-foreground"
										>
											Board token
										</label>
										<input
											id="new-board-token"
											class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
											placeholder="e.g. acmecorp"
											value={newValue()}
											onInput={(e) => setNewValue(e.currentTarget.value)}
											onKeyDown={(e) => e.key === "Enter" && handleAdd()}
										/>
									</div>
								</Show>

								{/* URL source: full search URL */}
								<Show when={currentSourceInfo()?.kind === "url"}>
									<div class="flex flex-col gap-2">
										<label
											for="new-search-url"
											class="text-xs font-medium text-foreground"
										>
											Search URL
										</label>
										<input
											id="new-search-url"
											class="rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
											placeholder={currentSourceInfo()?.url_prefix}
											value={newValue()}
											onInput={(e) => setNewValue(e.currentTarget.value)}
											onKeyDown={(e) => e.key === "Enter" && handleAdd()}
										/>
										<p class="text-xs text-faint">
											Must start with{" "}
											<span class="font-mono">
												{currentSourceInfo()?.url_prefix}
											</span>
										</p>
									</div>
								</Show>

								<label class="flex cursor-pointer items-center gap-2 text-xs font-medium text-foreground">
									<input
										type="checkbox"
										checked={newScrapeNow()}
										onChange={(e) => setNewScrapeNow(e.currentTarget.checked)}
										class="h-4 w-4 rounded border-border accent-primary"
									/>
									Scrape now — get results immediately instead of waiting up to
									6 hours
								</label>

								<div class="flex items-center gap-2 pt-1">
									<button
										type="button"
										onClick={handleAdd}
										disabled={createMutation.isPending || !newValue().trim()}
										class="rounded px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
									>
										{createMutation.isPending ? "Adding…" : "Add"}
									</button>
									<button
										type="button"
										onClick={() => {
											setShowAdd(false);
											resetForm();
										}}
										class="rounded px-3 py-1.5 text-xs font-medium text-muted transition hover:bg-surface-muted hover:text-foreground"
									>
										Cancel
									</button>
								</div>
							</div>
						</Show>
					</div>
				</Show>

				<Show when={!showAdd()}>
					<button
						type="button"
						onClick={() => {
							setScrapeQueued(false);
							setConflictError(null);
							setShowAdd(true);
						}}
						class="inline-flex items-center gap-1.5 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition hover:bg-primary-hover"
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
