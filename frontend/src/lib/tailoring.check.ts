import assert from "node:assert/strict";
import type {
	CVHeading,
	DraftFinding,
	SkillSuggestions,
	Suggestion,
} from "../types/tailoring";
import {
	allConfirmed,
	canExplain,
	isSettled,
	keptDraft,
	lineWithIds,
	moveItem,
	moveSuggestion,
	orderSuggestions,
	placeUnplaced,
	preselectedPicks,
	retarget,
	reviewFindings,
	selectedAchievementIds,
	setLinePicks,
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
const ranked = [
	suggestion("a", true),
	{ ...suggestion("x", true), positionId: "p2" },
	suggestion("b", true),
	suggestion("c", true),
];
assert.ok(
	moveSuggestion(ranked, "c", 0).join() === "c,x,a,b",
	"a move reorders only its own position's slots",
);
assert.ok(
	orderSuggestions(ranked, moveSuggestion(ranked, "b", 0))
		.map((s) => s.achievementId)
		.join() === "b,x,a,c",
	"the moved order round-trips through orderSuggestions",
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

assert.ok(!canExplain(suggestion("a", true)), "a bullet that fits has no Why?");
assert.ok(
	canExplain(suggestion("a", true, "low")),
	"a low bullet can be explained",
);
assert.ok(
	canExplain(suggestion("a", true, "unclear")),
	"an unclear bullet can be explained",
);

const skills: SkillSuggestions = {
	lines: [
		{
			label: "Languages",
			base: [
				{ text: "Go", score: 0.6, state: "fit" },
				{ text: "PHP", score: -0.4, state: "low" },
				{ text: "Perl", score: -0.1, state: "unclear" },
			],
			candidates: [
				{
					bankSkillId: "r",
					name: "Rust",
					score: 0.7,
					state: "fit",
					preselected: true,
					replaces: "PHP",
				},
				{
					bankSkillId: "e",
					name: "Elixir",
					score: 0.1,
					state: "fit",
					preselected: false,
					replaces: "",
				},
			],
		},
	],
	unplaced: [
		{
			bankSkillId: "k",
			name: "Kubernetes",
			score: 0.5,
			state: "fit",
			preselected: false,
			replaces: "",
		},
	],
};

const initial = preselectedPicks(skills);
assert.deepEqual(
	initial,
	[{ bankSkillId: "r", line: 0, replaces: "PHP" }],
	"preselected candidates start as picks with their server-chosen victim",
);
assert.deepEqual(
	setLinePicks(skills, initial, 0, ["r", "e"]).find(
		(p) => p.bankSkillId === "e",
	)?.replaces,
	"Perl",
	"a newly picked candidate replaces the lowest-lean item not already replaced",
);
assert.deepEqual(
	setLinePicks(skills, initial, 0, []),
	[],
	"clearing the line drops its picks",
);
assert.deepEqual(
	placeUnplaced(skills, initial, "k", 0).find((p) => p.bankSkillId === "k")
		?.replaces,
	"Perl",
	"an unplaced candidate takes the next free victim in the chosen line",
);
assert.deepEqual(
	placeUnplaced(skills, placeUnplaced(skills, initial, "k", 0), "k", undefined),
	initial,
	"unsetting the line removes the unplaced pick",
);
assert.equal(
	retarget(initial, "r", "Go")[0]?.replaces,
	"Go",
	"the replaced item can be changed",
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
