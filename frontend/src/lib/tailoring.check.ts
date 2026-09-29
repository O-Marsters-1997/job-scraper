import type { CVHeading, DraftFinding, Suggestion } from "../types/tailoring";
import {
	allConfirmed,
	isSettled,
	keptDraft,
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

function assert(cond: boolean, msg: string): void {
	if (!cond) throw new Error(msg);
}

assert(allConfirmed([]), "no headings counts as confirmed");
assert(
	allConfirmed([heading("A", true), heading("B", true)]),
	"all confirmed skips the mapping step",
);
assert(
	!allConfirmed([heading("A", true), heading("B", false)]),
	"one unconfirmed heading keeps the mapping step",
);

const mappings = toMappings([heading("A", false), heading("B", false)], {
	A: null,
});
assert(
	mappings[0]?.positionId === null,
	"a chosen none overrides the auto-match",
);
assert(
	mappings[1]?.positionId === "p1",
	"an untouched heading keeps its match",
);

const suggestion = (id: string, preselected: boolean): Suggestion => ({
	achievementId: id,
	positionId: "p1",
	text: id,
	score: 0.5,
	preselected,
});

const picked = selectedAchievementIds(
	[suggestion("a", true), suggestion("b", true), suggestion("c", false)],
	{ b: false, c: true },
);
assert(
	picked.join() === "a,c",
	"an override beats the preselection, either way",
);
assert(
	isSettled("ready") && isSettled("failed"),
	"ready and failed stop polling",
);
assert(
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
assert(
	reviewFindings(findings)
		.map((f) => f.message)
		.join() === "40%,long",
	"findings list blocking first and leave skill gaps out",
);
assert(
	skillGaps(findings).join() === "Terraform",
	"info-level skills findings are the skill gaps",
);
assert(
	keptDraft([{ outcome: null }, { outcome: "kept" }])?.outcome === "kept",
	"the kept draft is found among a job's drafts",
);
assert(
	keptDraft([{ outcome: "discarded" }]) === undefined,
	"a discarded draft is not kept",
);
