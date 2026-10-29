import assert from "node:assert/strict";
import { scoringPreviewSchema } from "../types/scoringPreview";

const parsed = scoringPreviewSchema.parse({
	dismissedInTop5: { before: 3, after: 1 },
	jobs: [
		{
			jobId: "j1",
			title: "Engineer",
			company: "Acme",
			before: 60,
			after: 72,
			afterBand: "good",
			afterRows: [],
		},
	],
});
assert.ok(parsed.jobs[0]?.grade === undefined, "grade is optional");
assert.ok(parsed.dismissedInTop5.after === 1, "summary parses");

assert.ok(
	!scoringPreviewSchema.safeParse({ dismissedInTop5: { before: 1, after: 1 } })
		.success,
	"missing jobs rejected",
);
