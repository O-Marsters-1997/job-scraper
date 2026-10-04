import type { Grade, GradeInput, GradeReason, GradeValue } from "@/types/grade";

export function planSaves(
	jobIds: string[],
	batch: GradeValue | undefined,
	overrides: Record<string, GradeValue>,
	reasons: Record<string, GradeReason[]>,
): GradeInput[] {
	return jobIds.flatMap((jobId) => {
		const grade = overrides[jobId] ?? batch;
		return grade ? [{ jobId, grade, reasons: reasons[jobId] ?? [] }] : [];
	});
}

export function undoPlan(
	saved: string[],
	priors: Record<string, Grade | null>,
): { restore: GradeInput[]; clear: string[] } {
	const restore: GradeInput[] = [];
	const clear: string[] = [];
	for (const jobId of saved) {
		const prior = priors[jobId];
		if (prior) {
			restore.push({ jobId, grade: prior.grade, reasons: prior.reasons });
		} else {
			clear.push(jobId);
		}
	}
	return { restore, clear };
}
