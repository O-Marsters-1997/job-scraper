import {
	type OutreachDraft,
	type OutreachDraftQuery,
	type OutreachHookInput,
	outreachDraftSchema,
} from "../types/outreach";
import { apiFetch, jsonInit } from "./client";
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

export async function writeOutreachHook(
	contactId: string,
	input: OutreachHookInput,
): Promise<OutreachDraft> {
	return mocked(
		(db) => db.writeOutreachHook(contactId, input),
		() =>
			apiFetch(
				`/outreach-drafts/${encodeURIComponent(contactId)}/hook`,
				outreachDraftSchema,
				jsonInit("POST", { achievementId: input.achievementId ?? "" }),
			),
	);
}
