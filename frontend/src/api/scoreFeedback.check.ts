import assert from "node:assert/strict";
import {
	appendOverallFeedback,
	listScoreFeedback,
} from "../mocks/scoreFeedback";
import { scoreFeedbackSchema } from "../types/scoreFeedback";

const created = scoreFeedbackSchema.parse(
	appendOverallFeedback("too generous"),
);
assert.equal(created.kind, "overall", "mock append is an overall entry");

appendOverallFeedback("second");
const listed = scoreFeedbackSchema.array().parse(listScoreFeedback());
assert.deepEqual(
	listed.map((e) => e.reason),
	["second", "too generous"],
	"mock list is newest first",
);

assert.throws(
	() => scoreFeedbackSchema.parse({ ...created, kind: "nonsense" }),
	"unknown kind rejected",
);
