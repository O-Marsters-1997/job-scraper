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

export const companiesQueryOptions = queryOptions({
	queryKey: ["companies"],
	queryFn: fetchCompanies,
});

export function useCompanies() {
	return createQuery(() => companiesQueryOptions);
}

export function useCompanyBoards(id: () => string) {
	return createQuery(() => ({
		queryKey: ["company-boards", id()],
		queryFn: () => fetchCompanyBoards(id()),
		enabled: Boolean(id()),
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
				queryKey: ["company-boards", board.CompanyID],
			}),
	}));
}

export function useAddCompany() {
	const queryClient = useQueryClient();
	return createMutation(() => ({
		mutationFn: (payload: AddCompanyPayload) => addCompany(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["companies"] });
			queryClient.invalidateQueries({ queryKey: ["source-targets"] });
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
			queryClient.invalidateQueries({ queryKey: ["companies"] });
			queryClient.invalidateQueries({ queryKey: ["source-targets"] });
		},
	}));
}
