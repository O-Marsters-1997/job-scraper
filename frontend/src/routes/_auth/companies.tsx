import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import { createEffect, For, on, Show, untrack } from "solid-js";
import type { CompanyPoster, CompanySort } from "@/api/companies";
import { FavouriteStar } from "@/components/FavouriteStar";
import { PageHeading } from "@/components/PageHeading";
import { Pager } from "@/components/Pager";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
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
	type CompanyListState,
	companySearchPatch,
	parseCompanySearch,
} from "@/lib/companyFilters";
import { formatDate } from "@/lib/datetime";
import {
	COMPANY_PAGE_SIZE,
	companyPageQuery,
	useCompanyPages,
	useSetCompanyFavourite,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { useDebouncedTerm } from "../../hooks/useDebouncedTerm";
import { queryClient } from "../../lib/queryClient";
import type { Company } from "../../types/company";

export const Route = createFileRoute("/_auth/companies")({
	validateSearch: (raw: Record<string, unknown>): Partial<CompanyListState> =>
		parseCompanySearch(raw),
	loaderDeps: ({ search }) => ({ state: parseCompanySearch(search) }),
	loader: ({ deps }) => queryClient.prefetchQuery(companyPageQuery(deps.state)),
	component: CompaniesPage,
});

type PosterOption = { value: CompanyPoster | "all"; label: string };

const POSTER_OPTIONS: PosterOption[] = [
	{ value: "all", label: "All companies" },
	{ value: "agency", label: "Agencies" },
	{ value: "direct", label: "Direct employers" },
];

const SORT_OPTIONS: { value: CompanySort; label: string }[] = [
	{ value: "relevance", label: "Relevance" },
	{ value: "alphabetical", label: "Alphabetical" },
	{ value: "open_jobs", label: "Open jobs" },
];

function CompaniesPage() {
	const [term, search] = useDebouncedTerm();
	const urlSearch = Route.useSearch();
	const state = () => parseCompanySearch(urlSearch());
	const navigate = useNavigate({ from: "/companies" });
	const table = useCompanyPages(state, (patch) =>
		navigate({
			to: "/companies",
			search: (prev) => ({ ...prev, ...companySearchPatch(patch) }),
			replace: true,
		}),
	);
	const query = table.query;

	createEffect(
		on(term, (q) => table.setFilters({ q }), {
			defer: true,
		}),
	);

	const trackMutation = useSetCompanyTracking();
	const favouriteMutation = useSetCompanyFavourite();

	const handleToggle = (c: Company) => {
		trackMutation.mutate({ id: c.ID, enabled: !c.Tracked });
	};

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Companies"
				subtitle="Every company we've encountered. Track a company to follow its current and future boards."
			>
				<Link to="/companies/new" class="text-xs text-muted hover:underline">
					Review new
				</Link>
			</PageHeading>

			<div class="mb-4 flex items-center gap-4">
				<div class="max-w-xs flex-1">
					<Label for="companies-search" class="sr-only">
						Search companies
					</Label>
					<Input
						id="companies-search"
						placeholder="Search companies…"
						value={untrack(() => state().q)}
						onInput={(e) => search(e.currentTarget.value)}
					/>
				</div>
				<Switch
					checked={state().favourite}
					onChange={(favourite) => table.setFilters({ favourite })}
				>
					<div class="flex items-center gap-2">
						<SwitchControl>
							<SwitchThumb />
						</SwitchControl>
						<SwitchLabel>Favourites</SwitchLabel>
					</div>
				</Switch>
				<Switch
					checked={state().noBoard}
					onChange={(noBoard) => table.setFilters({ noBoard })}
				>
					<div class="flex items-center gap-2">
						<SwitchControl>
							<SwitchThumb />
						</SwitchControl>
						<SwitchLabel>No Board discovered</SwitchLabel>
					</div>
				</Switch>
				<Select
					options={POSTER_OPTIONS}
					optionValue="value"
					optionTextValue="label"
					value={
						POSTER_OPTIONS.find((o) => o.value === (state().poster ?? "all")) ??
						null
					}
					onChange={(opt) => {
						if (!opt) return;
						table.setFilters({
							poster: opt.value === "all" ? undefined : opt.value,
						});
					}}
					itemComponent={(itemProps) => (
						<SelectItem item={itemProps.item}>
							<SelectItemLabel>{itemProps.item.rawValue.label}</SelectItemLabel>
						</SelectItem>
					)}
				>
					<SelectTrigger aria-label="Filter companies by poster">
						<Select.Value<PosterOption>>
							{(state) => state.selectedOption()?.label ?? "All companies"}
						</Select.Value>
					</SelectTrigger>
					<SelectContent />
				</Select>
				<div class="ml-auto">
					<Select
						options={SORT_OPTIONS}
						optionValue="value"
						optionTextValue="label"
						value={SORT_OPTIONS.find((o) => o.value === state().sort) ?? null}
						onChange={(opt) => {
							if (opt) table.setFilters({ sort: opt.value });
						}}
						itemComponent={(itemProps) => (
							<SelectItem item={itemProps.item}>
								<SelectItemLabel>
									{itemProps.item.rawValue.label}
								</SelectItemLabel>
							</SelectItem>
						)}
					>
						<SelectTrigger aria-label="Sort companies">
							<Select.Value<(typeof SORT_OPTIONS)[number]>>
								{(state) => state.selectedOption()?.label ?? "Sort"}
							</Select.Value>
						</SelectTrigger>
						<SelectContent />
					</Select>
				</div>
			</div>

			<QueryBoundary query={query} fallbackRows={6}>
				{(data) => (
					<>
						<p class="mb-2 text-xs text-muted">
							{data().verifiedBoardCompanies} of {data().allCompanies} companies
							have a verified board
						</p>
						<Card class="overflow-hidden">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead class="w-10">
											<span class="sr-only">Favourite</span>
										</TableHead>
										<TableHead>Name</TableHead>
										<TableHead>ATS</TableHead>
										<TableHead>Jobs</TableHead>
										<TableHead>First seen</TableHead>
										<TableHead class="w-16">Tracked</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									<For each={data().items}>
										{(c) => (
											<TableRow>
												<TableCell>
													<FavouriteStar
														favourite={c.Favourite ?? false}
														hourly={c.Tracked && c.CheckIntervalMinutes === 60}
														companyName={c.Name}
														disabled={favouriteMutation.isPending}
														onToggle={() =>
															favouriteMutation.mutate({
																id: c.ID,
																favourite: !c.Favourite,
															})
														}
													/>
												</TableCell>
												<TableCell>
													<Link
														to="/companies/$id"
														params={{ id: c.ID }}
														class="font-medium text-foreground hover:underline"
													>
														{c.Name}
													</Link>
												</TableCell>
												<TableCell>
													<Show
														when={c.ATSSource}
														fallback={
															<span class="text-xs text-faint">discovery</span>
														}
													>
														{(source) => (
															<Badge variant="source">{source()}</Badge>
														)}
													</Show>
												</TableCell>
												<TableCell class="font-mono text-xs tabular-nums text-muted">
													{c.JobCount}
												</TableCell>
												<TableCell class="font-mono text-xs tabular-nums text-faint">
													{formatDate(c.FirstSeenAt)}
												</TableCell>
												<TableCell>
													<Switch
														checked={c.Tracked}
														onChange={() => handleToggle(c)}
														disabled={trackMutation.isPending}
													>
														<SwitchLabel class="sr-only">
															Track {c.Name}
														</SwitchLabel>
														<SwitchControl>
															<SwitchThumb />
														</SwitchControl>
													</Switch>
												</TableCell>
											</TableRow>
										)}
									</For>
								</TableBody>
							</Table>
							<Pager
								total={data().total}
								page={table.page()}
								pageSize={COMPANY_PAGE_SIZE}
								noun="companies"
								onPage={table.setPage}
							/>
						</Card>
					</>
				)}
			</QueryBoundary>
		</div>
	);
}
