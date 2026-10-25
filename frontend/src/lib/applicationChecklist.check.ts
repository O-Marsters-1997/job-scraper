import assert from "node:assert/strict";
import type { Draft } from "@/types/tailoring";
import {
	type ChecklistInput,
	checklistState,
	linkedDraft,
} from "./applicationChecklist";

const base: ChecklistInput = {
	statusSystemKey: "draft",
	hasKeptDraft: false,
	hasOpenDraft: false,
	hasLetter: false,
	hasDoneLetter: false,
	applyTaskOpen: true,
	chaseBy: null,
};

assert.deepEqual(checklistState(base), {
	cv: "todo",
	coverLetter: "todo",
	apply: "todo",
	followUp: "todo",
});

assert.equal(checklistState({ ...base, hasOpenDraft: true }).cv, "inProgress");
assert.equal(checklistState({ ...base, hasKeptDraft: true }).cv, "done");
assert.equal(
	checklistState({ ...base, hasOpenDraft: true, hasKeptDraft: true }).cv,
	"done",
);
assert.equal(
	checklistState({ ...base, statusSystemKey: "applied" }).cv,
	"done",
);
assert.equal(checklistState({ ...base, statusSystemKey: "" }).cv, "done");

assert.equal(
	checklistState({ ...base, hasLetter: true }).coverLetter,
	"inProgress",
);
assert.equal(
	checklistState({ ...base, hasLetter: true, hasDoneLetter: true }).coverLetter,
	"done",
);

assert.equal(
	checklistState({ ...base, statusSystemKey: "applied", applyTaskOpen: false })
		.apply,
	"done",
);
assert.equal(
	checklistState({ ...base, statusSystemKey: "applied", applyTaskOpen: true })
		.apply,
	"todo",
);
assert.equal(checklistState({ ...base, applyTaskOpen: false }).apply, "todo");

assert.equal(
	checklistState({
		...base,
		statusSystemKey: "applied",
		chaseBy: "2026-10-20T00:00:00Z",
	}).followUp,
	"todo",
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
