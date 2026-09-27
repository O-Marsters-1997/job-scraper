import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	type AddCompanyPayload,
	addCompany,
	addCompanyBoard,
	fetchCompanies,
	fetchCompanyBoards,
	setCompanyTracking,
} from "../api/companies";
import { keys } from "../api/keys";

export const companiesQueryOptions = queryOptions({
	queryKey: keys.companies.all,
	queryFn: fetchCompanies,
});

export function useCompanies() {
	return createQuery(() => companiesQueryOptions);
}

export function useCompanyBoards(
	id: () => string,
	refetchInterval: () => number | false = () => false,
) {
	return createQuery(() => ({
		queryKey: keys.companies.boards(id()),
		queryFn: () => fetchCompanyBoards(id()),
		enabled: Boolean(id()),
		refetchInterval: refetchInterval(),
	}));
}

export function useAddCompanyBoard() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({
			id,
			url,
			confirm,
		}: {
			id: string;
			url: string;
			confirm: boolean;
		}) => addCompanyBoard(id, url, confirm),
		onSuccess: (board) =>
			queryClient.invalidateQueries({
				queryKey: keys.companies.boards(board.CompanyID),
			}),
	}));
}

export function useAddCompany() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: AddCompanyPayload) => addCompany(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.companies.all });
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets });
		},
	}));
}

export function useSetCompanyTracking() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: ({
			id,
			enabled,
			checkIntervalMinutes,
		}: {
			id: string;
			enabled: boolean;
			checkIntervalMinutes?: number;
		}) => setCompanyTracking(id, enabled, checkIntervalMinutes),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: keys.companies.all });
			queryClient.invalidateQueries({ queryKey: keys.sourceTargets });
		},
	}));
}
