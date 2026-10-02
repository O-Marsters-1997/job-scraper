import assert from "node:assert/strict";
import type { CVHeading, DraftFinding, Suggestion } from "../types/tailoring";
import {
	allConfirmed,
	isSettled,
	keptDraft,
	moveSuggestion,
	orderSuggestions,
	reviewFindings,
	selectedAchievementIds,
	skillGaps,
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

const suggestion = (id: string, preselected: boolean): Suggestion => ({
	achievementId: id,
	positionId: "p1",
	text: id,
	score: 0.5,
	state: "fit",
	preselected,
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
	"findings list blocking first and leave skill gaps out",
);
assert.ok(
	skillGaps(findings).join() === "Terraform",
	"info-level skills findings are the skill gaps",
);
assert.ok(
	keptDraft([{ outcome: null }, { outcome: "kept" }])?.outcome === "kept",
	"the kept draft is found among a job's drafts",
);
assert.ok(
	keptDraft([{ outcome: "discarded" }]) === undefined,
	"a discarded draft is not kept",
);
