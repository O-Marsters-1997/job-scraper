import assert from "node:assert/strict";
import type { Quota } from "../types/quota";
import { quotaLevel, worstLevel } from "./usageLevel";

const quota = (overrides: Partial<Quota>): Quota => ({
	provider: "decodo",
	status: "ok",
	used: 85,
	limit: 100,
	unit: "GB",
	percent: 85,
	level: "warn",
	resetsAt: null,
	fetchedAt: null,
	error: "",
	...overrides,
});

assert.equal(worstLevel([]), "ok");
assert.equal(worstLevel([undefined, "ok"]), "ok");
assert.equal(worstLevel(["ok", "warn"]), "warn");
assert.equal(worstLevel(["critical", "warn", "ok"]), "critical");
assert.equal(worstLevel([undefined, "warn", undefined]), "warn");

assert.equal(quotaLevel(undefined), "ok");
assert.equal(quotaLevel(quota({})), "warn");
assert.equal(quotaLevel(quota({ level: "critical" })), "critical");
assert.equal(quotaLevel(quota({ status: "not_configured" })), "ok");
assert.equal(quotaLevel(quota({ status: "error", level: "critical" })), "ok");
