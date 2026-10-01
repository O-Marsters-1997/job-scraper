import { KeptDraftExistsError } from "../lib/tailoring";
import {
	type CVHeading,
	cvHeadingSchema,
	type Draft,
	type DraftInput,
	type DraftRef,
	draftRefSchema,
	draftSchema,
	type HeadingMapping,
	headingMappingSchema,
	type SlotEdit,
	type Suggestion,
	suggestionSchema,
} from "../types/tailoring";
import { apiFetch, apiFetchBlob, jsonInit, rethrowStatus } from "./client";
import { mocked } from "./config";

export class MissingAiKeyError extends Error {
	constructor() {
		super("No OpenRouter key connected");
	}
}

const cvPath = (docId: string, tabId: string) =>
	`/tailoring/cvs/${encodeURIComponent(docId)}/${encodeURIComponent(tabId)}/headings`;

export async function fetchHeadings(
	docId: string,
	tabId: string,
): Promise<CVHeading[]> {
	return mocked(
		(db) => db.getHeadings(docId, tabId),
		() => apiFetch(cvPath(docId, tabId), cvHeadingSchema.array()),
	);
}

export async function saveHeadings(
	docId: string,
	tabId: string,
	mappings: HeadingMapping[],
): Promise<HeadingMapping[]> {
	return mocked(
		(db) => db.saveHeadings(docId, tabId, mappings),
		() =>
			apiFetch(
				cvPath(docId, tabId),
				headingMappingSchema.array(),
				jsonInit("PUT", { mappings }),
			),
	);
}

export async function fetchSuggestions(
	jobId: string,
	docId: string,
	tabId: string,
): Promise<Suggestion[]> {
	return mocked(
		(db) => db.getSuggestions(),
		() =>
			apiFetch(
				`/tailoring/jobs/${jobId}/suggestions?${new URLSearchParams({ docId, tabId })}`,
				suggestionSchema.array(),
			).catch(rethrowStatus({ 422: () => new MissingAiKeyError() })),
	);
}

export async function createDraft(input: DraftInput): Promise<DraftRef> {
	return mocked(
		(db) => db.createDraft(input),
		() =>
			apiFetch("/tailoring/drafts", draftRefSchema, jsonInit("POST", input)),
	);
}

export async function fetchDraft(id: string): Promise<Draft> {
	return mocked(
		(db) => db.getDraft(id),
		() => apiFetch(`/tailoring/drafts/${id}`, draftSchema),
	);
}

export async function fetchDraftPdf(id: string): Promise<ArrayBuffer> {
	return mocked(
		(db) => db.getMockPdfBytes(),
		() => apiFetchBlob(`/tailoring/drafts/${id}/pdf`),
	);
}

export async function fetchJobDrafts(jobId: string): Promise<Draft[]> {
	return mocked(
		(db) => db.getJobDrafts(jobId),
		() => apiFetch(`/tailoring/jobs/${jobId}/drafts`, draftSchema.array()),
	);
}

export async function keepDraft(id: string): Promise<Draft> {
	return mocked(
		(db) => db.keepDraft(id),
		() =>
			apiFetch(`/tailoring/drafts/${id}/keep`, draftSchema, {
				method: "POST",
			}).catch(rethrowStatus({ 409: () => new KeptDraftExistsError() })),
	);
}

export async function discardDraft(id: string): Promise<Draft> {
	return mocked(
		(db) => db.discardDraft(id),
		() =>
			apiFetch(`/tailoring/drafts/${id}/discard`, draftSchema, {
				method: "POST",
			}),
	);
}

export async function saveDraftSlots(
	id: string,
	slots: SlotEdit[],
): Promise<Draft> {
	return mocked(
		(db) => db.saveDraftSlots(id, slots),
		() =>
			apiFetch(
				`/tailoring/drafts/${id}/slots`,
				draftSchema,
				jsonInit("PUT", { slots }),
			),
	);
}
