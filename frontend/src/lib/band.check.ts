import assert from "node:assert/strict";
import { jobScoreSchema } from "../types/scores";
import { gradeDirection } from "./band";

assert.equal(gradeDirection("no", "good"), "too high");
assert.equal(gradeDirection("no", "great"), "too high");
assert.equal(gradeDirection("no", "fair"), "about right");
assert.equal(gradeDirection("no", "poor"), "about right");
assert.equal(gradeDirection("ok", "great"), "too high");
assert.equal(gradeDirection("ok", "good"), "about right");
assert.equal(gradeDirection("ok", "fair"), "about right");
assert.equal(gradeDirection("ok", "poor"), "too low");
assert.equal(gradeDirection("great", "great"), "about right");
assert.equal(gradeDirection("great", "good"), "about right");
assert.equal(gradeDirection("great", "fair"), "too low");
assert.equal(gradeDirection("great", "poor"), "too low");

assert.equal(
	jobScoreSchema.parse({ jobId: "1", score: 70, band: "good", rows: [] }).band,
	"good",
);
