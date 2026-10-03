import assert from "node:assert/strict";
import { setDemoData } from "../lib/demoData";
import { revertCorrection, setCorrection } from "./corrections";
import { fetchAllJobs } from "./jobs";

setDemoData(true);
const jobs = await fetchAllJobs();
const job = jobs.find((j) => j.Breakdown?.some((r) => r.resolved === "yes"));
const row = job?.Breakdown?.find((r) => r.resolved === "yes");
assert.ok(job && row, "demo data has a row resolved yes");
const target = { jobId: job.ID, optionId: row.key };

const corrected = await setCorrection({ ...target, value: "no" });
const after = corrected.rows.find((r) => r.key === row.key);
assert.equal(after?.resolved, "no");
assert.equal(after?.corrected, true);

const reverted = await revertCorrection(target);
const restored = reverted.rows.find((r) => r.key === row.key);
assert.equal(restored?.resolved, "yes");
assert.ok(!restored?.corrected, "revert clears the correction");
