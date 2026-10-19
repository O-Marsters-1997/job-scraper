import {
	type LetterSkeleton,
	letterSkeletonSchema,
} from "../types/coverLetter";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchLetterSkeleton(): Promise<LetterSkeleton> {
	return mocked(
		(db) => db.getLetterSkeleton(),
		() => apiFetch("/tailoring/letter-skeleton", letterSkeletonSchema),
	);
}

export async function saveLetterSkeleton(
	input: LetterSkeleton,
): Promise<LetterSkeleton> {
	return mocked(
		(db) => db.saveLetterSkeleton(input),
		() =>
			apiFetch(
				"/tailoring/letter-skeleton",
				letterSkeletonSchema,
				jsonInit("PUT", input),
			),
	);
}
