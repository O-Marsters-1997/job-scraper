import { createQuery } from "@tanstack/solid-query";
import { fetchOutreachDraft, writeOutreachHook } from "../api/outreachDrafts";
import type { OutreachDraftQuery, OutreachHookInput } from "../types/outreach";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export function useOutreachDraft(
	contactId: () => string,
	query: () => OutreachDraftQuery,
) {
	return createQuery(() => ({
		queryKey: ["outreach-drafts", contactId(), query()] as const,
		queryFn: () => fetchOutreachDraft(contactId(), query()),
	}));
}

export function useWriteOutreachHook(contactId: () => string) {
	return useInvalidatingMutation(
		(input: OutreachHookInput) => writeOutreachHook(contactId(), input),
		() => [["outreach-drafts", contactId()]],
	);
}
