import {
	type Contact,
	type ContactCompany,
	type ContactEvent,
	type RelationshipStatus,
	relationshipStatus,
} from "@/types/contact";

export type CompanyFilter = "all" | "employer" | "agency";

export interface CompanyRow {
	key: string;
	name: string;
	company: ContactCompany | undefined;
	people: Contact[];
	status: RelationshipStatus;
	lastTouch: string | undefined;
}

export interface HistoryEntry extends ContactEvent {
	who: string;
}

const isAgency = (kind: string) =>
	kind === "agency" || kind === "likely_agency";

const statusRank = (s: RelationshipStatus) =>
	s === "chase_due" ? 0 : s === "in_conversation" ? 1 : 2;

function lastTouch(people: Contact[]): string | undefined {
	return people
		.flatMap((c) => c.history.map((e) => e.occurred_on))
		.sort()
		.at(-1);
}

export function companyRows(
	companies: ContactCompany[],
	contacts: Contact[],
	today: string,
): CompanyRow[] {
	const byName = new Map<string, Contact[]>();
	for (const c of contacts) {
		const name = c.company_name.trim();
		if (c.company_id || !name) continue;
		const key = name.toLowerCase();
		byName.set(key, [...(byName.get(key) ?? []), c]);
	}
	const linked = companies.map((company) => ({
		key: company.id,
		name: company.name,
		company,
		people: contacts.filter((c) => c.company_id === company.id),
	}));
	const typed = [...byName.entries()].map(([key, people]) => ({
		key: `name:${key}`,
		name: people[0]?.company_name.trim() ?? key,
		company: undefined,
		people,
	}));
	return [...linked, ...typed]
		.map((r) => ({
			...r,
			status: relationshipStatus(r.people, today),
			lastTouch: lastTouch(r.people),
		}))
		.sort(
			(a, b) =>
				statusRank(a.status) - statusRank(b.status) ||
				a.name.localeCompare(b.name),
		);
}

export function filterRows(rows: CompanyRow[], filter: CompanyFilter) {
	if (filter === "all") return rows;
	return rows.filter(
		(r) => isAgency(r.company?.kind ?? "") === (filter === "agency"),
	);
}

export function firstName(name: string): string {
	return name.split(" ")[0] || name;
}

export function combinedHistory(people: Contact[]): HistoryEntry[] {
	return people
		.flatMap((c) => c.history.map((e) => ({ ...e, who: firstName(c.name) })))
		.sort((a, b) => b.occurred_on.localeCompare(a.occurred_on));
}
