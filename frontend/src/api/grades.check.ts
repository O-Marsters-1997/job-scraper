import assert from "node:assert/strict";
import { clearGrade, getGrade, isDismissed, setGrade } from "../mocks/grades";
import { gradeSchema } from "../types/grade";

assert.equal(getGrade("j1"), null, "ungraded job has no grade");

setGrade({ jobId: "j1", grade: "ok", reasons: [] });
const regraded = gradeSchema.parse(
	setGrade({ jobId: "j1", grade: "no", reasons: ["role"] }),
);
assert.deepEqual(regraded.reasons, ["role"], "re-grading replaces the grade");
assert.equal(isDismissed("j1"), true, "a no grade dismisses the job");

clearGrade("j1");
assert.equal(getGrade("j1"), null, "clearing removes the grade");
assert.equal(isDismissed("j1"), false, "clearing restores the job");

const culture = gradeSchema.parse(
	setGrade({ jobId: "j2", grade: "no", reasons: ["culture"] }),
);
assert.deepEqual(culture.reasons, ["culture"], "culture reason persists");

assert.throws(
	() => gradeSchema.parse({ ...regraded, reasons: ["vibes"] }),
	"unknown reason rejected",
);
