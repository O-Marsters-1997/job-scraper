import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createJobComment,
	deleteJobComment,
	fetchJobComments,
	updateJobComment,
} from "../api/jobComments";
import { keys } from "../api/keys";
import type { CommentQuote } from "../types/jobComment";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const jobCommentsQueryOptions = (jobId: string) =>
	queryOptions({
		queryKey: keys.jobComments(jobId),
		queryFn: () => fetchJobComments(jobId),
	});

export function useJobComments(jobId: () => string) {
	return createQuery(() => jobCommentsQueryOptions(jobId()));
}

export function useCreateJobComment(jobId: () => string) {
	return useInvalidatingMutation(
		(c: { body: string; quote: CommentQuote }) =>
			createJobComment(jobId(), c.body, c.quote),
		(comment) => [keys.jobComments(comment.JobID)],
	);
}

export function useUpdateJobComment(jobId: () => string) {
	return useInvalidatingMutation(
		(v: { id: string; body: string }) => updateJobComment(v.id, v.body),
		() => [keys.jobComments(jobId())],
	);
}

export function useDeleteJobComment(jobId: () => string) {
	return useInvalidatingMutation(
		(id: string) => deleteJobComment(id),
		() => [keys.jobComments(jobId())],
	);
}
