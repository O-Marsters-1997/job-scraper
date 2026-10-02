import { createQuery } from "@tanstack/solid-query";
import type { Accessor } from "solid-js";
import { keys } from "../api/keys";
import {
	createDraft,
	discardDraft,
	fetchDraft,
	fetchDraftLayout,
	fetchHeadings,
	fetchJobDrafts,
	fetchSuggestions,
	keepDraft,
	saveDraftSlots,
	saveHeadings,
} from "../api/tailoring";
import { isSettled } from "../lib/tailoring";
import type { HeadingMapping, SlotEdit } from "../types/tailoring";
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

export function useDraftLayout(id: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.draftLayout(id()),
		queryFn: () => fetchDraftLayout(id()),
		retry: false,
		refetchOnWindowFocus: false,
		staleTime: 0,
		gcTime: 0,
	}));
}

export function useJobDrafts(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.jobDrafts(jobId()),
		queryFn: () => fetchJobDrafts(jobId()),
		retry: false,
	}));
}

export function useKeepDraft() {
	return useInvalidatingMutation(keepDraft, [keys.tailoring.all]);
}

export function useDiscardDraft() {
	return useInvalidatingMutation(discardDraft, [keys.tailoring.all]);
}

export function useSaveDraftSlots() {
	return useInvalidatingMutation(
		({ id, slots }: { id: string; slots: SlotEdit[] }) =>
			saveDraftSlots(id, slots),
		[keys.tailoring.all],
	);
}
