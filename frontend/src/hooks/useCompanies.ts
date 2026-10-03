import {
	createInfiniteQuery,
	createQuery,
	keepPreviousData,
	queryOptions,
} from "@tanstack/solid-query";
import {
	addCompany,
	addCompanyBoard,
	excludeCompany,
	fetchCompany,
	fetchCompanyBoards,
	fetchCompanyPage,
	fetchNewCompanies,
	fetchTrackedCompanies,
	setCompanyFavourite,
	setCompanyReview,
	setCompanyTracking,
	unexcludeCompany,
} from "../api/companies";
import { keys } from "../api/keys";
import type { CompanyBoard, ReviewState } from "../types/company";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useCompanyPages(
	q: () => string,
	favourite: () => boolean = () => false,
) {
	return createInfiniteQuery(() => ({
		queryKey: keys.companies.list(q(), favourite()),
		queryFn: ({ pageParam }) =>
			fetchCompanyPage({
				q: q(),
				favourite: favourite(),
				cursor: pageParam,
			}),
		initialPageParam: "",
		getNextPageParam: (last) => last.next_cursor || undefined,
		placeholderData: keepPreviousData,
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
	);
}

const excludeInvalidations = [
	keys.companies.all,
	keys.jobs.all,
	keys.scoringConfig,
];

export function useExcludeCompany() {
	return useInvalidatingMutation(excludeCompany, excludeInvalidations);
}

export function useUnexcludeCompany() {
	return useInvalidatingMutation(
		({ id, removeName }: { id: string; removeName: boolean }) =>
			unexcludeCompany(id, removeName),
		excludeInvalidations,
	);
}

export function useSetCompanyReview() {
	return useInvalidatingMutation(
		({ id, state }: { id: string; state: ReviewState }) =>
			setCompanyReview(id, state),
		[keys.companies.all, keys.companies.tracked, keys.companies.new],
	);
}
