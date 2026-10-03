import assert from "node:assert/strict";
import type { Grade } from "../types/grade";
import { planSaves, undoPlan } from "./gradeBatch";

assert.deepEqual(
	planSaves(["a", "b", "c"], "ok", { b: "no" }, { b: ["role"] }),
	[
		{ jobId: "a", grade: "ok", reasons: [] },
		{ jobId: "b", grade: "no", reasons: ["role"] },
		{ jobId: "c", grade: "ok", reasons: [] },
	],
);

assert.deepEqual(
	planSaves(["a", "b"], undefined, { b: "great" }, { a: ["tech"] }),
	[{ jobId: "b", grade: "great", reasons: [] }],
);

const prior = (jobId: string, grade: Grade["grade"]): Grade => ({
	jobId,
	grade,
	reasons: ["salary"],
	updatedAt: "2026-10-03T00:00:00Z",
});

assert.deepEqual(
	undoPlan(["a", "b"], {
		a: prior("a", "great"),
		b: null,
		c: prior("c", "no"),
	}),
	{
		restore: [{ jobId: "a", grade: "great", reasons: ["salary"] }],
		clear: ["b"],
	},
);
