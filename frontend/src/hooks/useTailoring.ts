import { createQuery } from "@tanstack/solid-query";
import type { Accessor } from "solid-js";
import { keys } from "../api/keys";
import {
	fetchHeadings,
	fetchSuggestions,
	saveHeadings,
} from "../api/tailoring";
import type { HeadingMapping } from "../types/tailoring";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export type CVRef = { docId: string; tabId: string };

export function useHeadings(cv: Accessor<CVRef | undefined>) {
	return createQuery(() => {
		const ref = cv();
		const { docId = "", tabId = "" } = ref ?? {};
		return {
			queryKey: keys.tailoring.headings(docId, tabId),
			queryFn: () => fetchHeadings(docId, tabId),
			enabled: ref !== undefined,
		};
	});
}

export function useSaveHeadings() {
	return useInvalidatingMutation(
		({ docId, tabId, mappings }: CVRef & { mappings: HeadingMapping[] }) =>
			saveHeadings(docId, tabId, mappings),
		() => [keys.tailoring.all],
	);
}

export function useSuggestions(
	jobId: Accessor<string>,
	cv: Accessor<CVRef | undefined>,
) {
	return createQuery(() => {
		const ref = cv();
		const { docId = "", tabId = "" } = ref ?? {};
		return {
			queryKey: keys.tailoring.suggestions(jobId(), docId, tabId),
			queryFn: () => fetchSuggestions(jobId(), docId, tabId),
			enabled: ref !== undefined,
			retry: false,
			staleTime: 5 * 60 * 1000,
		};
	});
}
