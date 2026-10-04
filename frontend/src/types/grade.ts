import { z } from "zod";

export const GRADES = ["great", "ok", "no"] as const;
export type GradeValue = (typeof GRADES)[number];

export const GRADE_LABEL: Record<GradeValue, string> = {
	great: "Great",
	ok: "OK",
	no: "No",
};

export const GRADE_REASONS = [
	"seniority",
	"role",
	"tech",
	"domain",
	"location",
	"salary",
	"contract",
	"company_size",
	"recruiter",
	"culture",
	"other",
] as const;
export type GradeReason = (typeof GRADE_REASONS)[number];

export const gradeSchema = z.object({
	jobId: z.string(),
	grade: z.enum(GRADES),
	reasons: z.array(z.enum(GRADE_REASONS)),
	scoreAtGrade: z.number().optional(),
	scoreModel: z.string().optional(),
	updatedAt: z.string(),
});

export type Grade = z.infer<typeof gradeSchema>;

export interface GradeInput {
	jobId: string;
	grade: GradeValue;
	reasons: GradeReason[];
}
