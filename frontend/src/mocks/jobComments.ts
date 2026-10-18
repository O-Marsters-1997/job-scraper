import type { CommentQuote, JobComment } from "@/types/jobComment";

let comments: JobComment[] = [];
let nextId = 1;

export function getJobComments(jobId: string): JobComment[] {
	return comments.filter((c) => c.JobID === jobId).reverse();
}

export function createJobComment(
	jobId: string,
	body: string,
	quote: CommentQuote,
): JobComment {
	const now = new Date().toISOString();
	const comment: JobComment = {
		ID: `comment-${nextId++}`,
		JobID: jobId,
		Body: body,
		Quote: quote,
		CreatedAt: now,
		UpdatedAt: now,
	};
	comments = [...comments, comment];
	return comment;
}

export function updateJobComment(id: string, body: string): JobComment {
	const comment = comments.find((c) => c.ID === id);
	if (!comment) throw new Error("comment not found");
	const updated = {
		...comment,
		Body: body,
		UpdatedAt: new Date().toISOString(),
	};
	comments = comments.map((c) => (c.ID === id ? updated : c));
	return updated;
}

export function deleteJobComment(id: string): void {
	comments = comments.filter((c) => c.ID !== id);
}
