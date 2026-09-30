import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	addTrackedDoc,
	fetchCVTemplates,
	hideTab,
	showTab,
} from "../api/cvTemplates";
import { keys } from "../api/keys";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const cvTemplatesQueryOptions = queryOptions({
	queryKey: keys.cvTemplates,
	queryFn: fetchCVTemplates,
});

export function useCVTemplates() {
	return createQuery(() => cvTemplatesQueryOptions);
}

export function useAddTrackedDoc() {
	return useInvalidatingMutation(addTrackedDoc, [keys.cvTemplates]);
}

export function useHideTab() {
	return useInvalidatingMutation(
		({ docId, tabId }: { docId: string; tabId: string }) =>
			hideTab(docId, tabId),
		[keys.cvTemplates],
	);
}

export function useShowTab() {
	return useInvalidatingMutation(
		({ docId, tabId }: { docId: string; tabId: string }) =>
			showTab(docId, tabId),
		[keys.cvTemplates],
	);
}
