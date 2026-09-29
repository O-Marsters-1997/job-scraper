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
	type Suggestion,
	suggestionSchema,
} from "../types/tailoring";
import { apiFetch } from "./client";
import { mockDelay, mocked } from "./config";

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
		async (db) => {
			await mockDelay();
			return db.getHeadings(docId, tabId);
		},
		() => apiFetch(cvPath(docId, tabId), undefined, cvHeadingSchema.array()),
	);
}

export async function saveHeadings(
	docId: string,
	tabId: string,
	mappings: HeadingMapping[],
): Promise<HeadingMapping[]> {
	return mocked(
		async (db) => {
			await mockDelay(80);
			return db.saveHeadings(docId, tabId, mappings);
		},
		() =>
			apiFetch(
				cvPath(docId, tabId),
				{
					method: "PUT",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ mappings }),
				},
				headingMappingSchema.array(),
			),
	);
}

export async function fetchSuggestions(
	jobId: string,
	docId: string,
	tabId: string,
): Promise<Suggestion[]> {
	return mocked(
		async (db) => {
			await mockDelay(300);
			return db.getSuggestions();
		},
		async () => {
			const query = new URLSearchParams({ docId, tabId });
			try {
				return await apiFetch(
					`/tailoring/jobs/${jobId}/suggestions?${query}`,
					undefined,
					suggestionSchema.array(),
				);
			} catch (err) {
				if (err instanceof Error && err.message.endsWith(": 422")) {
					throw new MissingAiKeyError();
				}
				throw err;
			}
		},
	);
}

export async function createDraft(input: DraftInput): Promise<DraftRef> {
	return mocked(
		async (db) => {
			await mockDelay(120);
			return db.createDraft(input);
		},
		() =>
			apiFetch(
				"/tailoring/drafts",
				{
					method: "POST",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify(input),
				},
				draftRefSchema,
			),
	);
}

export async function fetchDraft(id: string): Promise<Draft> {
	return mocked(
		async (db) => {
			await mockDelay(60);
			return db.getDraft(id);
		},
		() => apiFetch(`/tailoring/drafts/${id}`, undefined, draftSchema),
	);
}
