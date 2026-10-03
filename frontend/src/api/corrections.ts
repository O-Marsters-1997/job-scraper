import {
	type CorrectionTarget,
	type CorrectionValue,
	type JobScore,
	jobScoreSchema,
} from "../types/scores";
import { apiFetch, jsonInit } from "./client";
import { mocked } from "./config";

const path = ({ jobId, optionId }: CorrectionTarget) =>
	`/jobs/${encodeURIComponent(jobId)}/corrections/${encodeURIComponent(optionId)}`;

export async function setCorrection(
	target: CorrectionTarget & { value: CorrectionValue },
): Promise<JobScore> {
	return mocked(
		(db) => db.correctAnswer(target, target.value),
		() =>
			apiFetch(
				path(target),
				jobScoreSchema,
				jsonInit("PUT", { value: target.value }),
			),
	);
}

export async function revertCorrection(
	target: CorrectionTarget,
): Promise<JobScore> {
	return mocked(
		(db) => db.correctAnswer(target, null),
		() => apiFetch(path(target), jobScoreSchema, { method: "DELETE" }),
	);
}
