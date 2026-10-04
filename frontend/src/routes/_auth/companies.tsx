import { createFileRoute, Link } from "@tanstack/solid-router";
import { createEffect, createSignal, For, Show } from "solid-js";
import type { CompanySort } from "@/api/companies";
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
import { formatDate } from "@/lib/datetime";
import {
	COMPANY_PAGE_SIZE,
	useCompanyPages,
	useSetCompanyFavourite,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { useDebouncedTerm } from "../../hooks/useDebouncedTerm";
import type { Company } from "../../types/company";

export const Route = createFileRoute("/_auth/companies")({
	component: CompaniesPage,
});

const SORT_OPTIONS: { value: CompanySort; label: string }[] = [
	{ value: "relevance", label: "Relevance" },
	{ value: "alphabetical", label: "Alphabetical" },
];

function CompaniesPage() {
	const [term, search] = useDebouncedTerm();
	const [favouritesOnly, setFavouritesOnly] = createSignal(false);
	const [noBoardOnly, setNoBoardOnly] = createSignal(false);
	const [sort, setSort] = createSignal<CompanySort>("relevance");
	const [page, setPage] = createSignal(1);
	const query = useCompanyPages(() => ({
		q: term(),
		favourite: favouritesOnly(),
		noBoard: noBoardOnly(),
		sort: sort(),
		offset: (page() - 1) * COMPANY_PAGE_SIZE,
	}));

	createEffect(() => {
		const total = query.data?.total;
		if (total === undefined || query.isPlaceholderData) return;
		const lastPage = Math.max(1, Math.ceil(total / COMPANY_PAGE_SIZE));
		if (page() > lastPage) setPage(lastPage);
	});

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
						onInput={(e) => {
							search(e.currentTarget.value);
							setPage(1);
						}}
					/>
				</div>
				<Switch
					checked={favouritesOnly()}
					onChange={(checked) => {
						setFavouritesOnly(checked);
						setPage(1);
					}}
				>
					<div class="flex items-center gap-2">
						<SwitchControl>
							<SwitchThumb />
						</SwitchControl>
						<SwitchLabel>Favourites</SwitchLabel>
					</div>
				</Switch>
				<Switch
					checked={noBoardOnly()}
					onChange={(checked) => {
						setNoBoardOnly(checked);
						setPage(1);
					}}
				>
					<div class="flex items-center gap-2">
						<SwitchControl>
							<SwitchThumb />
						</SwitchControl>
						<SwitchLabel>No Board discovered</SwitchLabel>
					</div>
				</Switch>
				<div class="ml-auto">
					<Select
						options={SORT_OPTIONS}
						optionValue="value"
						optionTextValue="label"
						value={SORT_OPTIONS.find((o) => o.value === sort()) ?? null}
						onChange={(opt) => {
							if (!opt) return;
							setSort(opt.value);
							setPage(1);
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
								from={(page() - 1) * COMPANY_PAGE_SIZE + 1}
								to={(page() - 1) * COMPANY_PAGE_SIZE + data().items.length}
								total={data().total}
								page={page()}
								pageCount={Math.ceil(data().total / COMPANY_PAGE_SIZE)}
								noun="companies"
								onPage={setPage}
							/>
						</Card>
					</>
				)}
			</QueryBoundary>
		</div>
	);
}
