import assert from "node:assert/strict";
import { playgroundQueueSchema } from "../types/scoringPlayground";

const parsed = playgroundQueueSchema.parse({
	jobIds: ["a", "b"],
	bands: [{ band: "great", reviewed: 1, remaining: 2 }],
});
assert.equal(parsed.jobIds.length, 2, "queue parses");
assert.throws(
	() =>
		playgroundQueueSchema.parse({
			jobIds: [],
			bands: [{ band: "", reviewed: 0, remaining: 0 }],
		}),
	"empty band is rejected",
);
