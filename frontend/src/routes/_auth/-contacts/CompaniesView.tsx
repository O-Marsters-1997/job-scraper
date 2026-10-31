import { format } from "date-fns";
import { createSignal, For, Show } from "solid-js";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useApplications } from "@/hooks/useApplications";
import { formatChaseDate } from "@/lib/chase";
import {
	type CompanyFilter,
	type CompanyRow,
	combinedHistory,
	companyRows,
	filterRows,
} from "@/lib/contactCompanies";
import { cn } from "@/lib/utils";
import {
	type Contact,
	type ContactCompany,
	relationshipLabels,
} from "@/types/contact";
import { eventLabel } from "./ContactCard";

const filters: { key: CompanyFilter; label: string }[] = [
	{ key: "all", label: "All" },
	{ key: "employer", label: "Employers" },
	{ key: "agency", label: "Agencies" },
];

const kindLabels: Record<string, string> = {
	agency: "Agency",
	likely_agency: "Likely agency",
	direct: "Employer",
	likely_direct: "Likely employer",
	unclassified: "Unclassified",
};

const today = () => format(new Date(), "yyyy-MM-dd");

function CompanyPeople(props: {
	people: Contact[];
	onOpenPerson: (c: Contact) => void;
}) {
	return (
		<div class="flex flex-col divide-y divide-border">
			<For each={props.people}>
				{(c) => (
					<button
						type="button"
						onClick={() => props.onOpenPerson(c)}
						class="flex items-baseline justify-between gap-3 px-4 py-2 text-left text-sm hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
					>
						<span class="truncate font-medium text-foreground">{c.name}</span>
						<span class="truncate text-xs text-muted">{c.role}</span>
					</button>
				)}
			</For>
		</div>
	);
}

function CompanyDetail(props: {
	row: CompanyRow;
	onOpenPerson: (c: Contact) => void;
	onAdd: (company: { id: string | null; name: string }) => void;
}) {
	const apps = useApplications();
	const row = () => props.row;
	const applications = () =>
		(apps.data ?? []).filter((a) =>
			row().company?.application_ids.includes(a.ID),
		);
	const uncovered = () =>
		(row().company?.open_jobs ?? []).filter((j) => !j.has_contact);
	const history = () => combinedHistory(row().people);

	return (
		<div class="grid border-t border-border md:grid-cols-[minmax(0,1fr)_19rem]">
			<div class="flex min-w-0 flex-col">
				<Show
					when={row().people.length > 0}
					fallback={
						<p class="px-4 py-3 text-sm text-muted">
							No one at {row().name} yet.
						</p>
					}
				>
					<CompanyPeople
						people={row().people}
						onOpenPerson={props.onOpenPerson}
					/>
				</Show>
				<div class="flex flex-col gap-3 border-t border-border px-4 py-4">
					<Show when={applications().length > 0}>
						<div class="flex flex-col gap-1">
							<span class="text-xs text-faint">Applications</span>
							<For each={applications()}>
								{(a) => (
									<span class="text-sm text-foreground">
										{a.JobTitle}
										<Show when={a.ChaseBy}>
											{(day) => (
												<span class="text-xs text-faint">
													{" "}
													· chase {formatChaseDate(day())}
												</span>
											)}
										</Show>
									</span>
								)}
							</For>
						</div>
					</Show>
					<Show when={uncovered().length > 0}>
						<div class="flex flex-col gap-1">
							<span class="text-xs text-faint">Open jobs with no contact</span>
							<For each={uncovered()}>
								{(j) => <span class="text-sm text-muted">{j.title}</span>}
							</For>
						</div>
					</Show>
					<div>
						<Button
							size="sm"
							variant="secondary"
							onClick={() =>
								props.onAdd({
									id: row().company?.id ?? null,
									name: row().name,
								})
							}
						>
							Add someone at {row().name}
						</Button>
					</div>
				</div>
			</div>
			<div class="flex flex-col gap-2 border-t border-border p-4 md:border-t-0 md:border-l">
				<span class="text-xs font-medium text-faint">History</span>
				<Show
					when={history().length > 0}
					fallback={<p class="text-sm text-faint">Nothing yet.</p>}
				>
					<ol class="flex flex-col gap-1 text-xs text-muted">
						<For each={history()}>
							{(e) => (
								<li>
									<span class="text-faint">
										{formatChaseDate(e.occurred_on)}
									</span>{" "}
									{e.who}: {eventLabel(e)}
								</li>
							)}
						</For>
					</ol>
				</Show>
			</div>
		</div>
	);
}

function openJobsSummary(company: ContactCompany | undefined): string {
	const open = company?.open_jobs ?? [];
	if (open.length === 0) return "";
	const none = open.filter((j) => !j.has_contact).length;
	const noun = open.length === 1 ? "open job" : "open jobs";
	return `${open.length} ${noun}${none > 0 ? ` · ${none} with no contact` : ""}`;
}

export function CompaniesView(props: {
	companies: ContactCompany[];
	contacts: Contact[];
	onOpenPerson: (c: Contact) => void;
	onAdd: (company: { id: string | null; name: string }) => void;
}) {
	const [filter, setFilter] = createSignal<CompanyFilter>("all");
	const [openKey, setOpenKey] = createSignal<string>();
	const rows = () =>
		filterRows(companyRows(props.companies, props.contacts, today()), filter());

	return (
		<div class="flex flex-col gap-3">
			<fieldset class="flex gap-1">
				<legend class="sr-only">Company type</legend>
				<For each={filters}>
					{(f) => (
						<button
							type="button"
							aria-pressed={filter() === f.key}
							onClick={() => setFilter(f.key)}
							class={cn(
								"rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
								filter() === f.key
									? "bg-accent-subtle text-accent-text"
									: "text-muted hover:bg-surface-muted",
							)}
						>
							{f.label}
						</button>
					)}
				</For>
			</fieldset>
			<Show
				when={rows().length > 0}
				fallback={
					<p class="text-sm text-faint">
						No companies yet. Track a company or add a contact to see it here.
					</p>
				}
			>
				<Card class="divide-y divide-border overflow-hidden">
					<For each={rows()}>
						{(row) => {
							const isOpen = () => openKey() === row.key;
							return (
								<div>
									<button
										type="button"
										aria-expanded={isOpen()}
										onClick={() => setOpenKey(isOpen() ? undefined : row.key)}
										class="flex w-full items-center gap-4 px-4 py-3 text-left hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
									>
										<span class="flex w-52 min-w-0 flex-col">
											<span class="truncate text-sm font-semibold text-foreground">
												{row.name}
											</span>
											<span class="text-xs text-faint">
												{row.company
													? (kindLabels[row.company.kind] ?? row.company.kind)
													: "Typed by name"}
											</span>
										</span>
										<span class="flex w-24 -space-x-2" aria-hidden="true">
											<For each={row.people.slice(0, 4)}>
												{(c) => (
													<span class="flex h-7 w-7 items-center justify-center rounded-full bg-surface-muted text-xs font-medium text-muted ring-2 ring-surface">
														{c.name.charAt(0).toUpperCase()}
													</span>
												)}
											</For>
										</span>
										<span class="hidden flex-1 text-xs text-muted md:block">
											{openJobsSummary(row.company)}
											<Show when={row.company?.application_ids.length}>
												{(n) => (
													<span class="text-faint">
														{" "}
														· {n()} application{n() === 1 ? "" : "s"}
													</span>
												)}
											</Show>
										</span>
										<span class="hidden w-28 text-right text-xs text-faint sm:block">
											{row.lastTouch ? formatChaseDate(row.lastTouch) : "-"}
										</span>
										<Badge
											variant={
												row.status === "chase_due" ? "default" : "secondary"
											}
										>
											{relationshipLabels[row.status]}
										</Badge>
									</button>
									<Show when={isOpen()}>
										<CompanyDetail
											row={row}
											onOpenPerson={props.onOpenPerson}
											onAdd={props.onAdd}
										/>
									</Show>
								</div>
							);
						}}
					</For>
				</Card>
			</Show>
		</div>
	);
}
