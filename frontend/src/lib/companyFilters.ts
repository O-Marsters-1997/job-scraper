import type { CompanyPoster, CompanySort } from "@/api/companies";

export interface CompanyListState {
	q: string;
	favourite: boolean;
	noBoard: boolean;
	sort: CompanySort;
	poster?: CompanyPoster | undefined;
	page?: number | undefined;
}

export const DEFAULT_COMPANY_STATE: CompanyListState = {
	q: "",
	favourite: false,
	noBoard: false,
	sort: "relevance",
};

export function parseCompanySearch(
	raw: Record<string, unknown>,
): CompanyListState {
	const page = Number(raw.page);
	return {
		q: typeof raw.q === "string" ? raw.q : "",
		favourite: raw.favourite === true || raw.favourite === "true",
		noBoard: raw.noBoard === true || raw.noBoard === "true",
		sort: raw.sort === "alphabetical" ? "alphabetical" : "relevance",
		poster:
			raw.poster === "agency" || raw.poster === "direct"
				? raw.poster
				: undefined,
		page: Number.isInteger(page) && page > 1 ? page : undefined,
	};
}

export function companySearchPatch(
	patch: Partial<CompanyListState>,
): Record<string, unknown> {
	return Object.fromEntries(
		Object.entries(patch).map(([key, value]) => [
			key,
			value === false || value === "" || value === DEFAULT_COMPANY_STATE.sort
				? undefined
				: value,
		]),
	);
}
