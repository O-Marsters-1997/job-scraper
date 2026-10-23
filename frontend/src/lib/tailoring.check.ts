import assert from "node:assert/strict";
import type { DraftFinding } from "../types/tailoring";
import {
	isSettled,
	keptDraft,
	lineWithIds,
	moveItem,
	reviewFindings,
} from "./tailoring";

assert.ok(
	isSettled("ready") && isSettled("failed"),
	"ready and failed stop polling",
);
assert.ok(
	!isSettled("pending") && !isSettled("running"),
	"in-flight keeps polling",
);

const finding = (
	check: string,
	severity: DraftFinding["severity"],
	message: string,
): DraftFinding => ({ check, severity, message });

const findings = [
	finding("length", "warn", "long"),
	finding("skills", "info", "Terraform"),
	finding("grounding", "block", "40%"),
];
assert.ok(
	reviewFindings(findings)
		.map((f) => f.message)
		.join() === "40%,long",
	"findings list blocking first and leave legacy skill gaps out",
);
assert.ok(
	keptDraft([{ outcome: null }, { outcome: "kept" }])?.outcome === "kept",
	"the kept draft is found among a job's drafts",
);
assert.ok(
	keptDraft([{ outcome: "discarded" }]) === undefined,
	"a discarded draft is not kept",
);

assert.deepEqual(
	lineWithIds(["Go", "SQL"], ["SQL", "Rust", "Go"]),
	["Go", "SQL", "Rust"],
	"a line keeps its order and appends what was added",
);
assert.deepEqual(
	lineWithIds(["Go", "SQL"], ["Go"]),
	["Go"],
	"a removed item leaves the line",
);
assert.deepEqual(
	moveItem(["Go", "SQL", "Rust"], 2, 0),
	["Rust", "Go", "SQL"],
	"an item moves up",
);
assert.deepEqual(
	moveItem(["Go", "SQL"], 0, -1),
	["Go", "SQL"],
	"a move off the end is ignored",
);
