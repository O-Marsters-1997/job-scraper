import {
	createQuery,
	keepPreviousData,
	queryOptions,
} from "@tanstack/solid-query";
import {
	addCompany,
	addCompanyBoard,
	excludeCompanies,
	fetchCompany,
	fetchCompanyAtsCounts,
	fetchCompanyBoards,
	fetchCompanyPage,
	fetchNewCompanies,
	fetchTrackedCompanies,
	setCompaniesKind,
	setCompanyFavourite,
	setCompanyKind,
	setCompanyReview,
	setCompanyTracking,
	unexcludeCompanies,
} from "../api/companies";
import { keys } from "../api/keys";
import type { CompanyListState } from "../lib/companyFilters";
import type { CompanyBoard, ReviewState } from "../types/company";
import { patchJobPages } from "./patchJobPages";
import { useInvalidatingMutation } from "./useInvalidatingMutation";
import { useServerTable } from "./useServerTable";

export const COMPANY_PAGE_SIZE = 50;

export const companyPageQuery = (s: CompanyListState) => ({
	queryKey: keys.companies.list(s),
	queryFn: () =>
		fetchCompanyPage({
			poster: s.poster,
			q: s.q,
			favourite: s.favourite,
			noBoard: s.noBoard,
			ats: s.ats,
			sort: s.sort,
			offset: ((s.page ?? 1) - 1) * COMPANY_PAGE_SIZE,
			limit: COMPANY_PAGE_SIZE,
		}),
});

export function useCompanyPages(
	state: () => CompanyListState,
	onChange: (patch: Partial<CompanyListState>) => void,
) {
	return useServerTable({
		state,
		onChange,
		pageSize: COMPANY_PAGE_SIZE,
		query: companyPageQuery,
		rowId: (company) => company.ID,
	});
}

export function useCompanyAtsCounts() {
	return createQuery(() => ({
		queryKey: keys.companies.atsCounts,
		queryFn: fetchCompanyAtsCounts,
	}));
}

const PICKER_LIMIT = 20;

export function useCompanySearch(q: () => string) {
	return createQuery(() => ({
		queryKey: keys.companies.search(q()),
		queryFn: () => fetchCompanyPage({ q: q(), limit: PICKER_LIMIT }),
		select: (page) => page.items,
		placeholderData: keepPreviousData,
	}));
}

export const companyQueryOptions = (id: string) =>
	queryOptions({
		queryKey: keys.companies.detail(id),
		queryFn: () => fetchCompany(id),
	});

export function useCompany(id: () => string) {
	return createQuery(() => ({
		...companyQueryOptions(id()),
		enabled: Boolean(id()),
	}));
}

export function useTrackedCompanies(poll: () => boolean = () => false) {
	return createQuery(() => ({
		queryKey: keys.companies.tracked,
		queryFn: fetchTrackedCompanies,
		refetchInterval: poll() ? 4000 : false,
	}));
}

export const newCompaniesQueryOptions = queryOptions({
	queryKey: keys.companies.new,
	queryFn: fetchNewCompanies,
});

export function useNewCompanies() {
	return createQuery(() => newCompaniesQueryOptions);
}

export function useCompanyBoards(
	id: () => string,
	refetchInterval: (
		boards: CompanyBoard[] | undefined,
	) => number | false = () => false,
) {
	return createQuery(() => ({
		queryKey: keys.companies.boards(id()),
		queryFn: () => fetchCompanyBoards(id()),
		enabled: Boolean(id()),
		refetchInterval: (query) => refetchInterval(query.state.data),
	}));
}

export function useAddCompanyBoard() {
	return useInvalidatingMutation(
		({ id, url, confirm }: { id: string; url: string; confirm: boolean }) =>
			addCompanyBoard(id, url, confirm),
		(board) => [keys.companies.boards(board.CompanyID), keys.companies.tracked],
	);
}

export function useAddCompany() {
	return useInvalidatingMutation(addCompany, [
		keys.companies.all,
		keys.sourceTargets,
	]);
}

export function useSetCompanyTracking() {
	return useInvalidatingMutation(
		({
			id,
			enabled,
			checkIntervalMinutes,
		}: {
			id: string;
			enabled: boolean;
			checkIntervalMinutes?: number;
		}) => setCompanyTracking(id, enabled, checkIntervalMinutes),
		[keys.companies.all, keys.sourceTargets],
	);
}

export function useSetCompanyFavourite() {
	return useInvalidatingMutation(
		({ id, favourite }: { id: string; favourite: boolean }) =>
			setCompanyFavourite(id, favourite),
		[keys.companies.all, keys.jobs.all],
		{
			patch: (queryClient, _result, { id, favourite }) =>
				patchJobPages(queryClient, (job) => job.CompanyID === id, {
					CompanyFavourite: favourite,
				}),
		},
	);
}

export function useSetCompanyKind() {
	return useInvalidatingMutation(
		({ id, kind }: { id: string; kind: "agency" | "direct" }) =>
			setCompanyKind(id, kind),
		[keys.companies.all, keys.jobs.all],
	);
}

const excludeInvalidations = [
	keys.companies.all,
	keys.jobs.all,
	keys.scoringConfig,
];

export function useExcludeCompanies() {
	return useInvalidatingMutation(excludeCompanies, excludeInvalidations);
}

export function useUnexcludeCompanies() {
	return useInvalidatingMutation(unexcludeCompanies, excludeInvalidations);
}

export function useSetCompaniesKind() {
	return useInvalidatingMutation(
		({ ids, kind }: { ids: string[]; kind: "agency" | "direct" }) =>
			setCompaniesKind(ids, kind),
		[keys.companies.all, keys.jobs.all],
	);
}

export function useSetCompanyReview() {
	return useInvalidatingMutation(
		({ id, state }: { id: string; state: ReviewState }) =>
			setCompanyReview(id, state),
		[keys.companies.all, keys.companies.tracked, keys.companies.new],
	);
}
