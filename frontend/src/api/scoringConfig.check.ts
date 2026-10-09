import assert from "node:assert/strict";
import { scoringConfigSchema } from "../types/scoringConfig";

const base = {
	notifyThreshold: 70,
	maxJobAgeDays: 7,
	excludedTitleKeywords: ["java", "sales"],
	excludedCompanies: ["acme corp"],
	excludedLocations: ["united states"],
	requiredLocations: ["london"],
	requiredTitleKeywords: ["engineer"],
	preferences: {
		picks: [
			{
				optionId: "tech:go",
				stance: "nice",
				source: "manual",
				overridden: false,
			},
		],
		salaryFloor: null,
		preferenceText: "",
	},
	updatedAt: "2026-09-27T00:00:00Z",
};

const valid = scoringConfigSchema.parse(base);
assert.ok(valid.notifyThreshold === 70, "valid config parses");
assert.ok(valid.excludedTitleKeywords.length === 2, "exclusion lists parse");
assert.ok(valid.preferences.picks.length === 1, "picks parse");

let threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: 101 });
} catch {
	threw = true;
}
assert.ok(threw, "threshold > 100 rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: -1 });
} catch {
	threw = true;
}
assert.ok(threw, "threshold < 0 rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: 70.5 });
} catch {
	threw = true;
}
assert.ok(threw, "float threshold rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, maxJobAgeDays: 366 });
} catch {
	threw = true;
}
assert.ok(threw, "max job age > 365 rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, excludedTitleKeywords: [1] });
} catch {
	threw = true;
}
assert.ok(threw, "non-string exclusion entry rejected");

threw = false;
try {
	scoringConfigSchema.parse({
		...base,
		preferences: { picks: [{ optionId: "tech:go", stance: "nice" }] },
	});
} catch {
	threw = true;
}
assert.ok(threw, "a pick missing source is rejected");

threw = false;
try {
	scoringConfigSchema.parse({
		...base,
		preferences: {
			...base.preferences,
			salaryFloor: { amount: -1, currency: "GBP" },
		},
	});
} catch {
	threw = true;
}
assert.ok(threw, "a negative salary floor amount is rejected");

const withFloor = scoringConfigSchema.parse({
	...base,
	preferences: {
		...base.preferences,
		salaryFloor: { amount: 55000, currency: "GBP" },
	},
});
assert.ok(
	withFloor.preferences.salaryFloor?.amount === 55000,
	"a salary floor parses",
);

const withoutBackfillQueued = scoringConfigSchema.parse(base);
assert.ok(
	withoutBackfillQueued.backfillQueued === 0,
	"backfillQueued defaults to 0 when absent",
);

const withBackfillQueued = scoringConfigSchema.parse({
	...base,
	backfillQueued: 3,
});
assert.ok(withBackfillQueued.backfillQueued === 3, "backfillQueued parses");
