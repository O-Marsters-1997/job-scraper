import { createFileRoute, Link } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
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
import { formatDate } from "@/lib/datetime";
import {
	companiesQueryOptions,
	useCompanies,
	useSetCompanyTracking,
} from "../../hooks/useCompanies";
import { queryClient } from "../../lib/queryClient";
import type { Company } from "../../types/company";

export const Route = createFileRoute("/_auth/companies")({
	loader: () => queryClient.ensureQueryData(companiesQueryOptions),
	component: CompaniesPage,
});

function CompaniesPage() {
	const query = useCompanies();
	const trackMutation = useSetCompanyTracking();

	const [search, setSearch] = createSignal("");

	const filtered = createMemo(() => {
		const q = search().trim().toLowerCase();
		const all = query.data ?? [];
		if (!q) return all;
		return all.filter((c) => c.Name.toLowerCase().includes(q));
	});

	const handleToggle = (c: Company) => {
		trackMutation.mutate({ id: c.ID, enabled: !c.Tracked });
	};

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="Companies"
				subtitle="Every company we've encountered. Track a company to follow its current and future boards."
			/>

			<div class="mb-4 max-w-xs">
				<Label for="companies-search" class="sr-only">
					Search companies
				</Label>
				<Input
					id="companies-search"
					placeholder="Search companies…"
					value={search()}
					onInput={(e) => setSearch(e.currentTarget.value)}
				/>
			</div>

			<QueryBoundary query={query} fallbackRows={6}>
				{() => (
					<Card class="overflow-hidden">
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Name</TableHead>
									<TableHead>ATS</TableHead>
									<TableHead>Jobs</TableHead>
									<TableHead>First seen</TableHead>
									<TableHead class="w-16">Tracked</TableHead>
								</TableRow>
							</TableHeader>
							<TableBody>
								<For each={filtered()}>
									{(c) => (
										<TableRow>
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
					</Card>
				)}
			</QueryBoundary>
		</div>
	);
}
