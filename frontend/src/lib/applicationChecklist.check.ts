import assert from "node:assert/strict";
import { type ChecklistInput, checklistDone } from "./applicationChecklist";

const base: ChecklistInput = {
	statusSystemKey: "draft",
	hasKeptDraft: false,
	hasDoneLetter: false,
	applyTaskOpen: true,
	chaseBy: null,
};

assert.deepEqual(checklistDone(base), {
	cv: false,
	coverLetter: false,
	apply: false,
	followUp: false,
});

assert.equal(checklistDone({ ...base, hasKeptDraft: true }).cv, true);
assert.equal(checklistDone({ ...base, statusSystemKey: "applied" }).cv, true);
assert.equal(checklistDone({ ...base, statusSystemKey: "" }).cv, true);

assert.equal(checklistDone({ ...base, hasDoneLetter: true }).coverLetter, true);

assert.equal(
	checklistDone({ ...base, statusSystemKey: "applied", applyTaskOpen: false })
		.apply,
	true,
);
assert.equal(
	checklistDone({ ...base, statusSystemKey: "applied", applyTaskOpen: true })
		.apply,
	false,
);
assert.equal(checklistDone({ ...base, applyTaskOpen: false }).apply, false);

assert.equal(
	checklistDone({
		...base,
		statusSystemKey: "applied",
		chaseBy: "2026-10-20T00:00:00Z",
	}).followUp,
	false,
);
