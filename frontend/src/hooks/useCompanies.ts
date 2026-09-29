import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	addCompany,
	addCompanyBoard,
	fetchCompanies,
	fetchCompanyBoards,
	setCompanyTracking,
} from "../api/companies";
import { keys } from "../api/keys";
import type { AddCompanyPayload, CompanyBoard } from "../types/company";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const companiesQueryOptions = queryOptions({
	queryKey: keys.companies.all,
	queryFn: fetchCompanies,
});

export function useCompanies() {
	return createQuery(() => companiesQueryOptions);
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
		(board) => [keys.companies.boards(board.CompanyID)],
	);
}

export function useAddCompany() {
	return useInvalidatingMutation(
		(payload: AddCompanyPayload) => addCompany(payload),
		[keys.companies.all, keys.sourceTargets],
	);
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
