import { createQuery } from "@tanstack/solid-query";
import { type Accessor, createMemo } from "solid-js";
import { keys } from "../api/keys";
import {
	createDraft,
	discardDraft,
	explainAchievement,
	fetchDraft,
	fetchDraftLayout,
	fetchExperienceMatch,
	fetchHeadings,
	fetchJobDrafts,
	fetchSkillSuggestions,
	fetchSuggestions,
	keepDraft,
	saveDraftSlots,
	saveHeadings,
} from "../api/tailoring";
import { isSettled } from "../lib/tailoring";
import type { HeadingMapping, SkillGroup, SlotEdit } from "../types/tailoring";
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

export function useSkillSuggestions(
	jobId: Accessor<string>,
	cv: Accessor<CVRef | undefined>,
) {
	return createQuery(() => {
		const ref = cv();
		const { docId = "", tabId = "" } = ref ?? {};
		return {
			queryKey: keys.tailoring.skillSuggestions(jobId(), docId, tabId),
			queryFn: () => fetchSkillSuggestions(jobId(), docId, tabId),
			enabled: ref !== undefined,
			retry: false,
			staleTime: 5 * 60 * 1000,
		};
	});
}

export function useExperienceMatch(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.experienceMatch(jobId()),
		queryFn: () => fetchExperienceMatch(jobId()),
		retry: false,
		staleTime: 5 * 60 * 1000,
	}));
}

export function useExplainAchievement(jobId: Accessor<string>) {
	return useInvalidatingMutation(
		(achievementId: string) => explainAchievement(jobId(), achievementId),
		[],
	);
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
	const stableId = createMemo(id);
	return createQuery(() => ({
		queryKey: keys.tailoring.draftLayout(stableId()),
		queryFn: () => fetchDraftLayout(stableId()),
		retry: false,
		refetchOnWindowFocus: false,
		gcTime: 0,
	}));
}

export function useJobDrafts(jobId: Accessor<string>) {
	return createQuery(() => ({
		queryKey: keys.tailoring.jobDrafts(jobId()),
		queryFn: () => fetchJobDrafts(jobId()),
		retry: false,
		refetchInterval: (query) =>
			query.state.data?.some((d) => !isSettled(d.status))
				? DRAFT_POLL_MS
				: false,
	}));
}

export function useKeepDraft() {
	return useInvalidatingMutation(keepDraft, [keys.tailoring.all]);
}

export function useSaveDraftSlots(id: Accessor<string>) {
	return useInvalidatingMutation(
		(slots: SlotEdit[]) => saveDraftSlots(id(), slots),
		() => [keys.tailoring.draft(id())],
	);
}

export function useSaveDraftSkills(id: Accessor<string>) {
	return useInvalidatingMutation(
		(skills: SkillGroup[]) => saveDraftSlots(id(), [], skills),
		() => [keys.tailoring.draft(id()), keys.tailoring.draftLayout(id())],
	);
}

export function useDiscardDraft() {
	return useInvalidatingMutation(discardDraft, [keys.tailoring.all]);
}
