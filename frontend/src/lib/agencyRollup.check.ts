import assert from "node:assert/strict";
import { agencyRollup, isAgencyKind } from "./agencyRollup";

assert.deepEqual(agencyRollup(["a", "b"], {}), {
	roles: 2,
	contacted: 0,
	lastContactedOn: null,
	bestOutcome: null,
});

assert.deepEqual(
	agencyRollup(["a", "b", "c"], {
		a: { count: 2, last_contacted_on: "2026-10-01", best_outcome: "spoke" },
		b: { count: 1, last_contacted_on: "2026-10-09", best_outcome: "no_reply" },
		z: { count: 1, last_contacted_on: "2026-12-01", best_outcome: "submitted" },
	}),
	{
		roles: 3,
		contacted: 2,
		lastContactedOn: "2026-10-09",
		bestOutcome: "spoke",
	},
);

assert.equal(isAgencyKind("agency"), true);
assert.equal(isAgencyKind("likely_agency"), true);
assert.equal(isAgencyKind("direct"), false);
assert.equal(isAgencyKind(undefined), false);
