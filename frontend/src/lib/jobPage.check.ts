import assert from "node:assert/strict";
import type { Job } from "@/types/job";
import { patchJobs } from "./jobPage";

const page = {
	total: 2,
	items: [
		{ ID: "a", Seen: false },
		{ ID: "b", Seen: false },
	] as Job[],
};

const patched = patchJobs(page, (job) => job.ID === "a", { Seen: true });
assert.equal(patched.items[0]?.Seen, true);
assert.equal(patched.items[1]?.Seen, false);
assert.equal(patched.total, 2, "a patch never changes total");
assert.equal(page.items[0]?.Seen, false, "the input page is not mutated");

console.log("✓ jobPage checks passed");
