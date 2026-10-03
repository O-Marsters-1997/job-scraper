import { type Grade, type GradeInput, gradeSchema } from "../types/grade";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

const gradePath = (jobId: string) => `/jobs/${encodeURIComponent(jobId)}/grade`;

export async function fetchGrade(jobId: string): Promise<Grade | null> {
	return mocked(
		(db) => db.getGrade(jobId),
		() => apiFetch(gradePath(jobId), gradeSchema.nullable()),
	);
}

export async function setGrade(input: GradeInput): Promise<Grade> {
	return mocked(
		(db) => db.setGrade(input),
		() =>
			apiFetch(
				gradePath(input.jobId),
				gradeSchema,
				jsonInit("PUT", { grade: input.grade, reasons: input.reasons }),
			),
	);
}

export async function clearGrade(jobId: string): Promise<void> {
	return mocked(
		(db) => db.clearGrade(jobId),
		() => apiFetchVoid(gradePath(jobId), { method: "DELETE" }),
	);
}
