import {
	type CommentQuote,
	type JobComment,
	jobCommentSchema,
} from "../types/jobComment";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchJobComments(jobId: string): Promise<JobComment[]> {
	return mocked(
		(db) => db.getJobComments(jobId),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/comments`,
				jobCommentSchema.array(),
			),
	);
}

export async function createJobComment(
	jobId: string,
	body: string,
	quote: CommentQuote,
): Promise<JobComment> {
	return mocked(
		(db) => db.createJobComment(jobId, body, quote),
		() =>
			apiFetch(
				`/jobs/${encodeURIComponent(jobId)}/comments`,
				jobCommentSchema,
				jsonInit("POST", { body, quote }),
			),
	);
}

export async function updateJobComment(
	id: string,
	body: string,
): Promise<JobComment> {
	return mocked(
		(db) => db.updateJobComment(id, body),
		() =>
			apiFetch(
				`/comments/${encodeURIComponent(id)}`,
				jobCommentSchema,
				jsonInit("PATCH", { body }),
			),
	);
}

export async function deleteJobComment(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteJobComment(id),
		() =>
			apiFetchVoid(`/comments/${encodeURIComponent(id)}`, {
				method: "DELETE",
			}),
	);
}
