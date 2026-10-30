import { createQuery } from "@tanstack/solid-query";
import { fetchOutreachDraft } from "../api/outreachDrafts";
import type { OutreachDraftQuery } from "../types/outreach";

export function useOutreachDraft(
	contactId: () => string,
	query: () => OutreachDraftQuery,
) {
	return createQuery(() => ({
		queryKey: ["outreach-drafts", contactId(), query()] as const,
		queryFn: () => fetchOutreachDraft(contactId(), query()),
	}));
}
