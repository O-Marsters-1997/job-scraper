import assert from "node:assert/strict";
import type { CVHeading, DraftFinding, Suggestion } from "../types/tailoring";
import {
	allConfirmed,
	isSettled,
	keptDraft,
	lineWithIds,
	moveItem,
	orderSuggestions,
	reviewFindings,
	selectedAchievementIds,
	toMappings,
} from "./tailoring";

const heading = (text: string, confirmed: boolean): CVHeading => ({
	text,
	positionId: "p1",
	confirmed,
	slotCount: 2,
});

assert.ok(allConfirmed([]), "no headings counts as confirmed");
assert.ok(
	allConfirmed([heading("A", true), heading("B", true)]),
	"all confirmed skips the mapping step",
);
assert.ok(
	!allConfirmed([heading("A", true), heading("B", false)]),
	"one unconfirmed heading keeps the mapping step",
);

const mappings = toMappings([heading("A", false), heading("B", false)], {
	A: null,
});
assert.ok(
	mappings[0]?.positionId === null,
	"a chosen none overrides the auto-match",
);
assert.ok(
	mappings[1]?.positionId === "p1",
	"an untouched heading keeps its match",
);

const suggestion = (
	id: string,
	preselected: boolean,
	state: Suggestion["state"] = "fit",
): Suggestion => ({
	achievementId: id,
	positionId: "p1",
	text: id,
	score: 0.5,
	preselected,
	state,
});

const picked = selectedAchievementIds(
	[suggestion("a", true), suggestion("b", true), suggestion("c", false)],
	{ b: false, c: true },
	[],
);
assert.ok(
	picked.join() === "a,c",
	"an override beats the preselection, either way",
);
assert.ok(
	selectedAchievementIds(
		[suggestion("a", true), suggestion("b", true), suggestion("c", true)],
		{},
		["c", "a"],
	).join() === "c,a,b",
	"chosen order comes first and unordered suggestions keep their rank",
);
assert.ok(
	orderSuggestions([suggestion("a", true), suggestion("b", true)], ["b"])
		.map((s) => s.achievementId)
		.join() === "b,a",
	"a chosen order puts its ids first",
);
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
