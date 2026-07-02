import {
	createMutation,
	createQuery,
	queryOptions,
	useQueryClient,
} from "@tanstack/solid-query";
import {
	type AddCompanyPayload,
	addCompany,
	fetchCompanies,
	setCompanyTracking,
} from "../api/companies";

export const companiesQueryOptions = queryOptions({
	queryKey: ["companies"],
	queryFn: fetchCompanies,
});

export function useCompanies() {
	return createQuery(() => companiesQueryOptions);
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
		mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
			setCompanyTracking(id, enabled),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["companies"] });
			queryClient.invalidateQueries({ queryKey: ["source-targets"] });
		},
	}));
}
