import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	addTrackedDoc,
	fetchCVTemplates,
	hideTab,
	showTab,
} from "../api/cvTemplates";
import { keys } from "../api/keys";
import type { TrackedDocKind } from "../types/cv";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const cvTemplatesQueryOptions = (kind: TrackedDocKind = "cv") =>
	queryOptions({
		queryKey: [...keys.cvTemplates, kind],
		queryFn: () => fetchCVTemplates(kind),
	});

export function useCVTemplates(kind: () => TrackedDocKind = () => "cv") {
	return createQuery(() => cvTemplatesQueryOptions(kind()));
}

export function useAddTrackedDoc(kind: () => TrackedDocKind = () => "cv") {
	return useInvalidatingMutation(
		(url: string) => addTrackedDoc(url, kind()),
		[keys.cvTemplates],
	);
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
