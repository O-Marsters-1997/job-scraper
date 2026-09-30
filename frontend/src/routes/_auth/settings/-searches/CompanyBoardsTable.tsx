import { Link } from "@tanstack/solid-router";
import { createMemo, For, Show } from "solid-js";
import { Pager } from "@/components/Pager";
import { SearchField } from "@/components/SearchField";
import { SourceBadge } from "@/components/SourceBadge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { paginate, relativeTime, type SearchParams } from "@/lib/searchTargets";
import {
	boardCounts,
	COMPANY_PAGE_SIZE,
	matchesCompany,
	sortCompanies,
} from "@/lib/trackedCompanies";
import { cn } from "@/lib/utils";
import type { TrackedBoard, TrackedCompany } from "../../../../types/company";
import type { SourceInfo } from "../../../../types/source";
import {
	DeleteIconButton,
	EmptyRow,
	EnabledSwitch,
	FilterChips,
	SortHead,
} from "./parts";

const ALL = "all";

export function CompanyBoardsTable(props: {
	companies: TrackedCompany[];
	sources: SourceInfo[];
	params: SearchParams;
	onParams: (
		patch: Partial<SearchParams>,
		opts?: { replace?: boolean },
	) => void;
	onToggle: (company: TrackedCompany) => void;
	onUntrack: (company: TrackedCompany) => void;
	onConfirm: (company: TrackedCompany, board: TrackedBoard) => void;
	confirming: (boardId: string) => boolean;
}) {
	const label = (name: string) =>
		props.sources.find((s) => s.name === name)?.label ?? name;

	const filtered = createMemo(() =>
		sortCompanies(
			props.companies.filter((c) => matchesCompany(c, props.params)),
			props.params.sort,
			props.params.dir,
		),
	);
	const paged = createMemo(() =>
		paginate(filtered(), props.params.page, COMPANY_PAGE_SIZE),
	);
	const filtersActive = () => Boolean(props.params.q || props.params.src);

	const sourceOptions = () => {
		const counts = boardCounts(props.companies);
		return [
			{ value: ALL, label: "All boards" },
			...props.sources
				.filter((s) => s.role === "ats" && counts.has(s.name))
				.map((s) => ({
					value: s.name,
					label: s.label,
					count: counts.get(s.name) ?? 0,
				})),
		];
	};

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-wrap items-center gap-2">
				<SearchField
					id="company-search"
					label="Search tracked companies"
					placeholder="Search companies…"
					value={props.params.q ?? ""}
					onInput={(q) =>
						props.onParams(
							{ q: q || undefined, page: undefined },
							{ replace: true },
						)
					}
				/>
				<FilterChips
					label="ATS"
					value={props.params.src ?? ALL}
					onChange={(v) =>
						props.onParams({ src: v === ALL ? undefined : v, page: undefined })
					}
					options={sourceOptions()}
				/>
				<Show when={filtersActive()}>
					<button
						type="button"
						onClick={() =>
							props.onParams({ q: undefined, src: undefined, page: undefined })
						}
						class="text-xs font-medium text-primary hover:underline"
					>
						Clear
					</button>
				</Show>
			</div>

			<Card class="overflow-hidden">
				<div class="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow class="hover:bg-transparent">
								<SortHead
									sortKey="company"
									firstDir="asc"
									params={props.params}
									onParams={props.onParams}
								>
									Company
								</SortHead>
								<TableHead>Boards</TableHead>
								<SortHead
									sortKey="open"
									firstDir="desc"
									params={props.params}
									onParams={props.onParams}
								>
									Roles
								</SortHead>
								<SortHead
									sortKey="checked"
									firstDir="desc"
									params={props.params}
									onParams={props.onParams}
								>
									Last checked
								</SortHead>
								<TableHead class="w-20">Active</TableHead>
								<TableHead class="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							<For
								each={paged().items}
								fallback={
									<EmptyRow colSpan={6}>
										{filtersActive()
											? "No companies match these filters."
											: "No tracked companies yet. Paste a company board URL above."}
									</EmptyRow>
								}
							>
								{(c) => {
									const unverified = () =>
										c.boards.find((b) => b.status === "candidate");
									return (
										<TableRow
											class={cn("[&>td]:py-2.5", !c.enabled && "text-faint")}
										>
											<TableCell>
												<Link
													to="/companies/$id"
													params={{ id: c.id }}
													class="max-w-64 truncate font-medium text-foreground hover:underline"
												>
													{c.name}
												</Link>
											</TableCell>
											<TableCell>
												<div class="flex flex-wrap items-center gap-1.5">
													<For each={c.boards}>
														{(b) => (
															<a
																href={b.url}
																target="_blank"
																rel="noreferrer"
																aria-label={`${label(b.source)} board for ${c.name}`}
																class="rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
															>
																<SourceBadge source={label(b.source)} />
															</a>
														)}
													</For>
													<Show when={unverified()}>
														{(board) => (
															<>
																<span class="whitespace-nowrap rounded bg-surface-muted px-1.5 py-0.5 text-xs text-muted">
																	Unverified — not polling yet
																</span>
																<Button
																	variant="outline"
																	size="sm"
																	disabled={props.confirming(board().id)}
																	onClick={() => props.onConfirm(c, board())}
																>
																	{props.confirming(board().id)
																		? "Confirming…"
																		: "Confirm board"}
																</Button>
															</>
														)}
													</Show>
												</div>
											</TableCell>
											<TableCell class="font-mono text-xs tabular-nums">
												<Link
													to="/jobs"
													search={{ company: c.id, scored: true }}
													class="text-primary hover:underline"
												>
													<strong>{c.relevant_jobs}</strong> of {c.open_jobs}{" "}
													open
												</Link>
											</TableCell>
											<TableCell class="whitespace-nowrap text-xs text-muted">
												{relativeTime(c.last_checked_at)}
											</TableCell>
											<TableCell>
												<EnabledSwitch
													enabled={c.enabled}
													name={c.name}
													onToggle={() => props.onToggle(c)}
												/>
											</TableCell>
											<TableCell>
												<DeleteIconButton
													label={`Untrack ${c.name}`}
													title="Untrack"
													onClick={() => props.onUntrack(c)}
												/>
											</TableCell>
										</TableRow>
									);
								}}
							</For>
						</TableBody>
					</Table>
				</div>
				<Pager
					from={paged().from}
					to={paged().to}
					total={paged().total}
					page={paged().page}
					pageCount={paged().pageCount}
					noun="companies"
					onPage={(page) =>
						props.onParams({ page: page > 1 ? page : undefined })
					}
				/>
			</Card>
		</div>
	);
}
