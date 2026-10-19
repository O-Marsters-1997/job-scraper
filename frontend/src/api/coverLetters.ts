import {
	type CoverLetter,
	coverLetterSchema,
	type LetterCheck,
	letterCheckSchema,
	type ScaffoldInput,
} from "../types/coverLetter";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchCoverLetters(jobId: string): Promise<CoverLetter[]> {
	return mocked(
		(db) => db.getCoverLetters(jobId),
		() =>
			apiFetch(
				`/tailoring/jobs/${encodeURIComponent(jobId)}/cover-letters`,
				coverLetterSchema.array(),
			),
	);
}

export async function checkCoverLetter(id: string): Promise<LetterCheck> {
	return mocked(
		(db) => db.checkCoverLetter(id),
		() =>
			apiFetch(
				`/tailoring/cover-letters/${encodeURIComponent(id)}/check`,
				letterCheckSchema,
			),
	);
}

export async function markCoverLetterDone(id: string): Promise<CoverLetter> {
	return mocked(
		(db) => db.markCoverLetterDone(id),
		() =>
			apiFetch(
				`/tailoring/cover-letters/${encodeURIComponent(id)}/done`,
				coverLetterSchema,
				{ method: "POST" },
			),
	);
}

export async function createCoverLetter(
	input: ScaffoldInput,
): Promise<CoverLetter> {
	const { jobId, ...body } = input;
	return mocked(
		(db) => db.createCoverLetter(input),
		() =>
			apiFetch(
				`/tailoring/jobs/${encodeURIComponent(jobId)}/cover-letters`,
				coverLetterSchema,
				jsonInit("POST", body),
			),
	);
}
