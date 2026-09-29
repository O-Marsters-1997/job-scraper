import { createQuery } from "@tanstack/solid-query";
import type { Accessor } from "solid-js";
import { keys } from "../api/keys";
import {
	createDraft,
	fetchDraft,
	fetchHeadings,
	fetchSuggestions,
	saveHeadings,
} from "../api/tailoring";
import { isSettled } from "../lib/tailoring";
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

export function useCreateDraft() {
	return useInvalidatingMutation(createDraft, []);
}

const DRAFT_POLL_MS = 2000;

export function useDraft(id: Accessor<string | undefined>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.draft(id() ?? ""),
		queryFn: () => fetchDraft(id() ?? ""),
		enabled: id() !== undefined,
		retry: false,
		refetchInterval: (query) =>
			query.state.data && isSettled(query.state.data.status)
				? false
				: DRAFT_POLL_MS,
	}));
}
