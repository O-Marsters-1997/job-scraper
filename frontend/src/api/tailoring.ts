import { createSseParser } from "../lib/sse";
import { KeptDraftExistsError } from "../lib/tailoring";
import {
	type CVHeading,
	cvHeadingSchema,
	type Draft,
	type DraftInput,
	type DraftLayout,
	type DraftRef,
	draftLayoutSchema,
	draftRefSchema,
	draftSchema,
	type HeadingMapping,
	headingMappingSchema,
	type SlotEdit,
	type SuggestDone,
	type Suggestion,
	type SuggestRequest,
	suggestDoneSchema,
	suggestionSchema,
} from "../types/tailoring";
import {
	ApiError,
	apiFetch,
	apiFetchBlob,
	jsonInit,
	request,
	rethrowStatus,
} from "./client";
import { mocked } from "./config";

export class MissingAiKeyError extends Error {
	constructor() {
		super("No OpenRouter key connected");
	}
}

export class LayoutUnsupportedError extends Error {}

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

export async function fetchDraftLayout(id: string): Promise<DraftLayout> {
	return mocked(
		(db) => db.getDraftLayout(),
		() =>
			apiFetch(`/tailoring/drafts/${id}/layout`, draftLayoutSchema).catch(
				(err) => {
					if (err instanceof ApiError && err.status === 422)
						throw new LayoutUnsupportedError(unsupportedReason(err.body));
					throw err;
				},
			),
	);
}

function unsupportedReason(body: unknown): string {
	const reason = (body as { error?: unknown } | undefined)?.error;
	return typeof reason === "string" ? reason : "unsupported layout";
}

const messageOf = (data: string) =>
	(JSON.parse(data) as { message?: string }).message ?? "Suggestion failed";

export async function streamSuggestion(
	id: string,
	slotId: string,
	req: SuggestRequest,
	onDelta: (text: string) => void,
	signal: AbortSignal,
): Promise<SuggestDone> {
	return mocked(
		(db) => db.streamSuggestion(req, onDelta, signal),
		async () => {
			const res = await request(
				`/tailoring/drafts/${id}/slots/${encodeURIComponent(slotId)}/suggest`,
				{ ...jsonInit("POST", req), signal },
			).catch((err) => {
				if (err instanceof ApiError) {
					const msg = (err.body as { error?: unknown } | undefined)?.error;
					if (typeof msg === "string") throw new Error(msg);
				}
				throw err;
			});
			if (!res.body) throw new Error("Suggestion failed");
			let done: SuggestDone | undefined;
			let failure: string | undefined;
			const feed = createSseParser((event, data) => {
				if (event === "delta")
					onDelta((JSON.parse(data) as { text: string }).text);
				if (event === "done") done = suggestDoneSchema.parse(JSON.parse(data));
				if (event === "error") failure = messageOf(data);
			});
			const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
			for (;;) {
				const chunk = await reader.read();
				if (chunk.done) break;
				feed(chunk.value);
			}
			if (failure !== undefined) throw new Error(failure);
			if (!done) throw new Error("Suggestion ended early");
			return done;
		},
	);
}
