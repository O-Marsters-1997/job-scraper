import type { Grade, GradeInput } from "../types/grade";
import { getJobs } from "./jobs";

const grades = new Map<string, Grade>();

export function getGrade(jobId: string): Grade | null {
	return structuredClone(grades.get(jobId) ?? null);
}

export function setGrade(input: GradeInput): Grade {
	const score = getJobs().find((j) => j.ID === input.jobId)?.SuitabilityScore;
	const grade: Grade = {
		...input,
		scoreAtGrade: score ?? undefined,
		updatedAt: new Date().toISOString(),
	};
	grades.set(input.jobId, grade);
	return structuredClone(grade);
}

export function clearGrade(jobId: string): void {
	grades.delete(jobId);
}

export function isDismissed(jobId: string): boolean {
	return grades.get(jobId)?.grade === "no";
}
