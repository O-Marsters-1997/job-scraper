import type { TrackedCompany } from "@/types/company";
import type { SearchParams } from "./searchTargets";

export const COMPANY_PAGE_SIZE = 25;

export function matchesCompany(
	company: TrackedCompany,
	params: SearchParams,
): boolean {
	const needle = params.q?.trim().toLowerCase();
	if (needle && !company.name.toLowerCase().includes(needle)) return false;
	if (params.src && !company.boards.some((b) => b.source === params.src))
		return false;
	return true;
}

export function sortCompanies(
	companies: TrackedCompany[],
	sort: SearchParams["sort"],
	dir: SearchParams["dir"] = "asc",
): TrackedCompany[] {
	const by = (a: TrackedCompany, b: TrackedCompany): number => {
		switch (sort) {
			case "company":
				return a.name.localeCompare(b.name);
			case "open":
				return a.open_jobs - b.open_jobs;
			case "checked":
				return (
					new Date(a.last_checked_at ?? 0).getTime() -
					new Date(b.last_checked_at ?? 0).getTime()
				);
			default:
				return 0;
		}
	};
	if (sort !== "company" && sort !== "open" && sort !== "checked")
		return companies;
	const sign = dir === "asc" ? 1 : -1;
	return [...companies].sort((a, b) => sign * by(a, b));
}

export function boardCounts(companies: TrackedCompany[]): Map<string, number> {
	const counts = new Map<string, number>();
	for (const c of companies) {
		for (const source of new Set(c.boards.map((b) => b.source)))
			counts.set(source, (counts.get(source) ?? 0) + 1);
	}
	return counts;
}
