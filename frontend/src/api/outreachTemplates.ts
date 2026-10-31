import {
	type OutreachTemplate,
	type OutreachTemplateInput,
	outreachTemplateSchema,
} from "../types/outreach";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchOutreachTemplates(): Promise<OutreachTemplate[]> {
	return mocked(
		(db) => db.getOutreachTemplates(),
		() => apiFetch("/outreach-templates", outreachTemplateSchema.array()),
	);
}

export async function createOutreachTemplate(
	input: OutreachTemplateInput,
): Promise<OutreachTemplate> {
	return mocked(
		(db) => db.createOutreachTemplate(input),
		() =>
			apiFetch(
				"/outreach-templates",
				outreachTemplateSchema,
				jsonInit("POST", input),
			),
	);
}

export async function updateOutreachTemplate(
	id: string,
	input: Pick<OutreachTemplateInput, "name" | "body">,
): Promise<OutreachTemplate> {
	return mocked(
		(db) => db.updateOutreachTemplate(id, input),
		() =>
			apiFetch(
				`/outreach-templates/${encodeURIComponent(id)}`,
				outreachTemplateSchema,
				jsonInit("PATCH", input),
			),
	);
}

export async function makeOutreachTemplateDefault(
	id: string,
): Promise<OutreachTemplate> {
	return mocked(
		(db) => db.makeOutreachTemplateDefault(id),
		() =>
			apiFetch(
				`/outreach-templates/${encodeURIComponent(id)}/default`,
				outreachTemplateSchema,
				{ method: "POST" },
			),
	);
}

export async function deleteOutreachTemplate(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteOutreachTemplate(id),
		() =>
			apiFetchVoid(`/outreach-templates/${encodeURIComponent(id)}`, {
				method: "DELETE",
			}),
	);
}
