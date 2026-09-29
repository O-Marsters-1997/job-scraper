import type { CVHeading, Suggestion } from "../types/tailoring";
import {
	allConfirmed,
	isSettled,
	selectedAchievementIds,
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
