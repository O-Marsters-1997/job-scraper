import assert from "node:assert/strict";
import { cvStage } from "./cvStage";

const draft = (
	Status: "pending" | "running" | "keeping" | "ready" | "failed",
	Outcome: "kept" | "discarded" | null = null,
) => ({ ID: "d", Status, Outcome, LastError: "" });

assert.equal(cvStage(null), "none");
assert.equal(cvStage(draft("pending")), "generating");
assert.equal(cvStage(draft("running")), "generating");
assert.equal(cvStage(draft("ready")), "inProgress");
assert.equal(cvStage(draft("keeping")), "completing");
assert.equal(cvStage(draft("failed")), "failed");
assert.equal(cvStage(draft("ready", "kept")), "complete");
