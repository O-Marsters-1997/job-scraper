import assert from "node:assert/strict";
import type { Draft } from "@/types/tailoring";
import {
	type ChecklistInput,
	checklistDone,
	linkedDraft,
} from "./applicationChecklist";

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

const draft = (id: string, over: Partial<Draft> = {}): Draft =>
	({ id, status: "ready", outcome: null, ...over }) as Draft;

assert.equal(linkedDraft([]), undefined);
assert.equal(
	linkedDraft([draft("open"), draft("kept", { outcome: "kept" })])?.id,
	"kept",
);
assert.equal(
	linkedDraft([
		draft("gone", { outcome: "discarded" }),
		draft("bad", { status: "failed" }),
	]),
	undefined,
);
assert.equal(linkedDraft([draft("p", { status: "pending" })])?.id, "p");
assert.equal(linkedDraft([draft("r", { status: "running" })])?.id, "r");
assert.equal(linkedDraft([draft("new"), draft("old")])?.id, "new");
