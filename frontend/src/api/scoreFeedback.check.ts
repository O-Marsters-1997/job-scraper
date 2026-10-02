import assert from "node:assert/strict";
import {
	appendJobFeedback,
	appendOverallFeedback,
	deleteScoreFeedback,
	listScoreFeedback,
} from "../mocks/scoreFeedback";
import {
	SCORE_FEEDBACK_PAGE_SIZE,
	scoreFeedbackPageSchema,
	scoreFeedbackSchema,
} from "../types/scoreFeedback";

const created = scoreFeedbackSchema.parse(
	appendOverallFeedback("too generous"),
);
assert.equal(created.kind, "overall", "mock append is an overall entry");

appendOverallFeedback("second");
const listed = scoreFeedbackPageSchema.parse(listScoreFeedback());
assert.deepEqual(
	listed.entries.map((e) => e.reason),
	["second", "too generous"],
	"mock list is newest first",
);
assert.equal(listed.total, 2, "mock list reports the total");
assert.equal(listScoreFeedback("job").total, 0, "mock list filters by kind");

for (let i = 0; i < SCORE_FEEDBACK_PAGE_SIZE; i++) appendOverallFeedback("x");
const second = listScoreFeedback(undefined, 2);
assert.equal(second.entries.length, 2, "mock list pages");
assert.equal(second.total, SCORE_FEEDBACK_PAGE_SIZE + 2);

deleteScoreFeedback(created.id);
assert.equal(
	listScoreFeedback().total,
	SCORE_FEEDBACK_PAGE_SIZE + 1,
	"mock delete removes the entry",
);

assert.throws(
	() => scoreFeedbackSchema.parse({ ...created, kind: "nonsense" }),
	"unknown kind rejected",
);

const job = scoreFeedbackSchema.parse(
	appendJobFeedback({ jobId: "missing", direction: "lower", reason: "high" }),
);
assert.equal(job.direction, "lower", "mock job entry keeps its direction");
