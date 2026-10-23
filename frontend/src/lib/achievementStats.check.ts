import assert from "node:assert/strict";
import type { Achievement, AchievementStats } from "@/types/experience";
import { sortAchievements, statLine } from "./achievementStats";

const stats = (
	inCvs: number,
	leftOut: number,
	applied: number,
	replied: number,
): AchievementStats => ({ inCvs, leftOut, applied, replied });

const item = (id: string, s: AchievementStats | null): Achievement => ({
	id,
	positionId: "p",
	text: id,
	context: "",
	stats: s,
});

assert.equal(statLine(null), "");
assert.equal(statLine(stats(0, 0, 0, 0)), "");
assert.equal(
	statLine(stats(0, 2, 0, 0)),
	"in 0 CVs · left out 2× while higher",
);
assert.equal(
	statLine(stats(3, 2, 2, 1)),
	"in 3 CVs · left out 2× while higher · 1/2 replied",
);
assert.equal(statLine(stats(1, 0, 0, 0)), "in 1 CV");
assert.equal(statLine(stats(9, 0, 5, 2)), "in 9 CVs · 2/5 replied (40%)");

const items = [
	item("none", null),
	item("busy", stats(9, 0, 4, 4)),
	item("rated-low", stats(2, 0, 5, 1)),
	item("rated-high", stats(1, 0, 5, 4)),
];
const ids = (xs: Achievement[]) => xs.map((x) => x.id);
assert.deepEqual(ids(sortAchievements(items, "bank")), ids(items));
assert.deepEqual(ids(sortAchievements(items, "usage")), [
	"busy",
	"rated-low",
	"rated-high",
	"none",
]);
assert.deepEqual(ids(sortAchievements(items, "rate")), [
	"rated-high",
	"rated-low",
	"busy",
	"none",
]);
