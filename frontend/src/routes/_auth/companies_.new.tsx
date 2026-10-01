import { createFileRoute, Link } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	newCompaniesQueryOptions,
	useNewCompanies,
	useSetCompanyReview,
} from "../../hooks/useCompanies";
import { queryClient } from "../../lib/queryClient";
import type { CompanyProfileEntry, NewCompany } from "../../types/company";

export const Route = createFileRoute("/_auth/companies_/new")({
	loader: () => queryClient.ensureQueryData(newCompaniesQueryOptions),
	component: NewCompaniesPage,
});

function NewCompaniesPage() {
	const query = useNewCompanies();
	const review = useSetCompanyReview();

	return (
		<div class="px-7 py-6">
			<PageHeading
				title="New companies"
				subtitle="Companies discovered for you. Keep the ones worth following; dismiss the rest."
			>
				<Link to="/companies" class="text-xs text-muted hover:underline">
					All companies
				</Link>
			</PageHeading>

			<QueryBoundary query={query} fallbackRows={4}>
				{(companies) => (
					<Show
						when={companies().length > 0}
						fallback={<p class="text-sm text-faint">Nothing new to review.</p>}
					>
						<div class="grid gap-3">
							<For each={companies()}>
								{(c) => (
									<NewCompanyCard
										company={c}
										pending={review.isPending}
										onReview={(state) => review.mutate({ id: c.id, state })}
									/>
								)}
							</For>
						</div>
					</Show>
				)}
			</QueryBoundary>
		</div>
	);
}

function NewCompanyCard(props: {
	company: NewCompany;
	pending: boolean;
	onReview: (state: "kept" | "dismissed") => void;
}) {
	return (
		<Card class="flex items-center justify-between gap-4 p-4">
			<div class="min-w-0">
				<Link
					to="/companies/$id"
					params={{ id: props.company.id }}
					class="font-medium text-foreground hover:underline"
				>
					{props.company.name}
				</Link>
				<div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs text-muted">
					<For each={props.company.boards}>
						{(b) => <Badge variant="source">{b.source}</Badge>}
					</For>
					<span class="tabular-nums">
						{props.company.matching_roles} matching{" "}
						{props.company.matching_roles === 1 ? "role" : "roles"}
					</span>
					<Show when={props.company.best_suitability !== null}>
						<span class="tabular-nums">
							Best suitability {props.company.best_suitability}
						</span>
					</Show>
				</div>
				<Show when={props.company.profile.length > 0}>
					<ul class="mt-1.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted">
						<For each={props.company.profile}>
							{(e) => (
								<li>
									{e.dimension}: {e.label}{" "}
									<span class="tabular-nums">{profileSummary(e)}</span>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</div>
			<div class="flex shrink-0 gap-2">
				<Button
					size="sm"
					disabled={props.pending}
					onClick={() => props.onReview("kept")}
				>
					Keep
				</Button>
				<Button
					size="sm"
					variant="outline"
					disabled={props.pending}
					onClick={() => props.onReview("dismissed")}
				>
					Dismiss
				</Button>
			</div>
		</Card>
	);
}

function profileSummary(e: CompanyProfileEntry) {
	if (e.known === 0) return "not known";
	return `${e.yes} of ${e.total} ${e.total === 1 ? "role" : "roles"}`;
}
