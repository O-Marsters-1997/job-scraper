import {
	type OutreachDraft,
	type OutreachDraftQuery,
	outreachDraftSchema,
} from "../types/outreach";
import { apiFetch } from "./client";
import { mocked } from "./config";

export async function fetchOutreachDraft(
	contactId: string,
	query: OutreachDraftQuery,
): Promise<OutreachDraft> {
	return mocked(
		(db) => db.getOutreachDraft(contactId, query),
		() => {
			const params = new URLSearchParams();
			if (query.templateId) params.set("templateId", query.templateId);
			if (query.achievementId) params.set("achievementId", query.achievementId);
			const qs = params.size > 0 ? `?${params}` : "";
			return apiFetch(
				`/outreach-drafts/${encodeURIComponent(contactId)}${qs}`,
				outreachDraftSchema,
			);
		},
	);
}
