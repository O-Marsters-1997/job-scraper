import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Icon } from "@/components/Icon";
import { QueryBoundary } from "@/components/QueryBoundary";
import { SettingsActions } from "@/components/SettingsLayout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { resolveBoard } from "../../../api/sources";
import { ConflictError } from "../../../api/sourceTargets";
import { useSources } from "../../../hooks/useSources";
import {
	useCreateSourceTarget,
	useDeleteSourceTarget,
	useRerunSourceTarget,
	useSourceTargets,
	useUpdateSourceTarget,
} from "../../../hooks/useSourceTargets";
import type { SourceInfo } from "../../../types/source";
import type { SourceTarget } from "../../../types/sourceTarget";

export const Route = createFileRoute("/_auth/settings/searches")({
	component: SearchesPage,
});

function SearchesPage() {
	const query = useSourceTargets();
	const sourcesQuery = useSources();
	const createMutation = useCreateSourceTarget();
	const updateMutation = useUpdateSourceTarget();
	const deleteMutation = useDeleteSourceTarget();
	const rerunMutation = useRerunSourceTarget();

	const [showAdd, setShowAdd] = createSignal(false);
	const [selectedSource, setSelectedSource] = createSignal("wis");
	const [newValue, setNewValue] = createSignal("");
	const [newFilters, setNewFilters] = createSignal<Record<string, string>>({});
	const [conflictError, setConflictError] = createSignal<string | null>(null);
	const [scrapeQueued, setScrapeQueued] = createSignal(false);
	const [pasteUrl, setPasteUrl] = createSignal("");
	const [resolveHint, setResolveHint] = createSignal<string | null>(null);
	const [resolving, setResolving] = createSignal(false);

	const currentSourceInfo = () =>
		(sourcesQuery.data ?? []).find((s) => s.name === selectedSource());

	const sourceLabel = (name: string) =>
		(sourcesQuery.data ?? []).find((s) => s.name === name)?.label ?? name;

	const sourceRole = (name: string) =>
		(sourcesQuery.data ?? []).find((s) => s.name === name)?.role ?? "discovery";

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
		addForm.setError(null);
		setNewValue("");
		setNewFilters({});
		setConflictError(null);
		setPasteUrl("");
		setResolveHint(null);
	};

	const handleResolve = async (event: Event) => {
		event.preventDefault();
		const url = pasteUrl().trim();
		if (!url) return;
		setResolveHint(null);
		setResolving(true);
		try {
			const r = await resolveBoard(url);
			if (!r) {
				setResolveHint(
					"Couldn't detect an ATS board from that URL — pick a source below instead.",
				);
				return;
			}
			setSelectedSource(r.source);
			setNewValue(r.value);
			setNewFilters({});
			setResolveHint(`Detected ${sourceLabel(r.source)} board "${r.value}".`);
		} catch {
			setResolveHint(
				"Couldn't resolve that URL — pick a source below instead.",
			);
		} finally {
			setResolving(false);
		}
	};

	const addForm = useFormSubmit(
		async () => {
			const info = currentSourceInfo();
			const val = newValue().trim();
			if (!val) return;

			if (info?.kind === "url" && !val.startsWith(info.url_prefix)) {
				setConflictError(`URL must start with ${info.url_prefix}`);
				return;
			}

			setConflictError(null);
			setScrapeQueued(false);
			const filters = info?.kind === "filter" ? newFilters() : {};
			const created = await createMutation.mutateAsync({
				source: selectedSource(),
				value: val,
				filters,
			});
			resetForm();
			setShowAdd(false);
			if (created.RunStatus === "failed") {
				setConflictError(
					"Search saved, but it could not start. Use Run again to retry.",
				);
			} else if (sourceRole(created.Source) === "discovery") {
				setScrapeQueued(true);
			}
		},
		(err) =>
			err instanceof ConflictError
				? "A search with these settings already exists."
				: "Failed to add search. Please try again.",
	);

	const handleToggle = (t: SourceTarget) => {
		updateMutation.mutate({ id: t.ID, enabled: !t.Enabled });
	};

	const handleDelete = (id: string) => {
		deleteMutation.mutate(id);
	};

	const handleRerun = async (id: string) => {
		setConflictError(null);
		setScrapeQueued(false);
		try {
			await rerunMutation.mutateAsync(id);
			setScrapeQueued(true);
		} catch {
			setConflictError("Could not start the search. Please try again.");
		}
	};

	const TargetsCard = (props: { list: SourceTarget[] }) => (
		<Card class="mb-4 overflow-x-auto">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Source</TableHead>
						<TableHead>Search</TableHead>
						<TableHead>Filters</TableHead>
						<TableHead>Last run</TableHead>
						<TableHead>Status</TableHead>
						<TableHead class="w-24" />
						<TableHead class="w-16" />
						<TableHead class="w-20" />
					</TableRow>
				</TableHeader>
				<TableBody>
					<For each={props.list}>
						{(t) => (
							<TableRow>
								<TableCell>
									<Badge variant="source">{sourceLabel(t.Source)}</Badge>
								</TableCell>
								<TableCell
									class="max-w-[200px] truncate font-mono text-xs"
									title={t.Value}
								>
									{t.Value}
								</TableCell>
								<TableCell
									class="max-w-[200px] truncate text-xs text-faint"
									title={filterSummary(t)}
								>
									{filterSummary(t)}
								</TableCell>
								<TableCell>
									{t.LastRunAt
										? new Date(t.LastRunAt).toLocaleString()
										: "Never"}
								</TableCell>
								<TableCell>
									<Show when={sourceRole(t.Source) === "discovery"}>
										<span title={t.LastRunError || undefined}>
											{t.RunStatus}
										</span>
									</Show>
								</TableCell>
								<TableCell>
									<Show when={sourceRole(t.Source) === "discovery"}>
										<Button
											variant="outline"
											size="sm"
											onClick={() => handleRerun(t.ID)}
											disabled={
												rerunMutation.isPending ||
												t.RunStatus === "queued" ||
												t.RunStatus === "running"
											}
										>
											Run again
										</Button>
									</Show>
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
										class="rounded px-2 py-1 text-xs font-medium text-destructive-strong transition hover:bg-destructive-subtle disabled:opacity-50"
									>
										Delete
									</button>
								</TableCell>
							</TableRow>
						)}
					</For>
				</TableBody>
			</Table>
		</Card>
	);

	return (
		<>
			<SettingsActions>
				<Show when={!showAdd()}>
					<Button
						size="sm"
						onClick={() => {
							setScrapeQueued(false);
							setConflictError(null);
							addForm.setError(null);
							setShowAdd(true);
						}}
					>
						<Icon name="plus" size={12} strokeWidth={2.5} />
						Add search
					</Button>
				</Show>
			</SettingsActions>
			<FormFeedback
				success={
					scrapeQueued()
						? "Scrape queued. Matching jobs will appear shortly."
						: false
				}
				error={conflictError() ?? addForm.error()}
			/>

			<QueryBoundary query={query} fallbackRows={3}>
				{(data) => (
					<>
						<Show when={showAdd()}>
							<div class="mb-4 rounded-xl border border-border bg-surface px-4 py-4">
								<Show
									when={sourcesQuery.isSuccess}
									fallback={<p class="text-sm text-muted">Loading sources…</p>}
								>
									<div class="flex flex-col gap-3">
										<form
											onSubmit={handleResolve}
											class="flex flex-col gap-2 rounded-lg bg-surface-muted px-3 py-3"
										>
											<label
												for="paste-url"
												class="text-xs font-medium text-foreground"
											>
												Paste a job-board URL
											</label>
											<div class="flex items-center gap-2">
												<Input
													id="paste-url"
													autofocus
													placeholder="e.g. https://boards.greenhouse.io/acmecorp"
													value={pasteUrl()}
													onInput={(e) => setPasteUrl(e.currentTarget.value)}
												/>
												<button
													type="submit"
													disabled={resolving() || !pasteUrl().trim()}
													class="shrink-0 rounded px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
												>
													{resolving() ? "Detecting…" : "Detect"}
												</button>
											</div>
											<Show when={resolveHint()}>
												<p class="text-xs text-faint">{resolveHint()}</p>
											</Show>
											<p class="text-xs text-faint">
												We'll detect the ATS and fill in the board below. Or
												pick a source manually.
											</p>
										</form>

										<form onSubmit={addForm.submit} class="flex flex-col gap-3">
											<div class="flex flex-col gap-2">
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
													<Select.Label class="block text-xs font-medium text-foreground">
														Source
													</Select.Label>
													<SelectTrigger>
														<Select.Value<SourceInfo>>
															{(state) =>
																state.selectedOption()?.label ??
																"Select a source"
															}
														</Select.Value>
													</SelectTrigger>
													<SelectContent />
												</Select>
											</div>

											<Show when={currentSourceInfo()?.kind === "filter"}>
												<div class="flex flex-col gap-2">
													<label
														for="new-keywords"
														class="text-xs font-medium text-foreground"
													>
														Keywords
													</label>
													<Input
														id="new-keywords"
														placeholder="e.g. product engineer"
														value={newValue()}
														onInput={(e) => setNewValue(e.currentTarget.value)}
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
															<Input
																id={`filter-${field.name}`}
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
															/>
														</div>
													)}
												</For>
											</Show>

											<Show when={currentSourceInfo()?.kind === "board"}>
												<div class="flex flex-col gap-2">
													<label
														for="new-board-token"
														class="text-xs font-medium text-foreground"
													>
														Board token
													</label>
													<Input
														id="new-board-token"
														placeholder="e.g. acmecorp"
														value={newValue()}
														onInput={(e) => setNewValue(e.currentTarget.value)}
													/>
												</div>
											</Show>

											<Show when={currentSourceInfo()?.kind === "url"}>
												<div class="flex flex-col gap-2">
													<label
														for="new-search-url"
														class="text-xs font-medium text-foreground"
													>
														Search URL
													</label>
													<Input
														id="new-search-url"
														placeholder={currentSourceInfo()?.url_prefix}
														value={newValue()}
														onInput={(e) => setNewValue(e.currentTarget.value)}
													/>
													<p class="text-xs text-faint">
														Must start with{" "}
														<span class="font-mono">
															{currentSourceInfo()?.url_prefix}
														</span>
													</p>
												</div>
											</Show>

											<div class="flex items-center gap-2 pt-1">
												<button
													type="submit"
													disabled={addForm.pending() || !newValue().trim()}
													class="rounded px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-accent-subtle disabled:opacity-50"
												>
													{addForm.pending() ? "Adding…" : "Add"}
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
										</form>
									</div>
								</Show>
							</div>
						</Show>
						<Show when={data().some((t) => sourceRole(t.Source) === "ats")}>
							<div class="mb-2 mt-1">
								<h2 class="text-sm font-semibold text-foreground">
									Tracked companies
								</h2>
								<p class="text-xs text-faint">
									ATS boards re-checked every few hours for new roles.
								</p>
							</div>
							<TargetsCard
								list={data().filter((t) => sourceRole(t.Source) === "ats")}
							/>
						</Show>

						<Show when={data().some((t) => sourceRole(t.Source) !== "ats")}>
							<div class="mb-2 mt-1">
								<h2 class="text-sm font-semibold text-foreground">
									Discovery searches
								</h2>
								<p class="text-xs text-faint">
									Keyword and URL searches across aggregators and job boards.
								</p>
							</div>
							<TargetsCard
								list={data().filter((t) => sourceRole(t.Source) !== "ats")}
							/>
						</Show>

						<p class="mt-3 text-xs text-faint">
							Discovery searches run when added or explicitly rerun.
						</p>
					</>
				)}
			</QueryBoundary>
		</>
	);
}
