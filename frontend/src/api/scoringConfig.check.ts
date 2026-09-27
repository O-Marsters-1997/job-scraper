import { scoringConfigSchema } from "../types/scoringConfig";

function ok(cond: boolean, msg: string) {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}

const base = {
	notifyThreshold: 70,
	excludedTitleKeywords: ["java", "sales"],
	excludedCompanies: ["acme corp"],
	excludedLocations: ["united states"],
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
ok(valid.notifyThreshold === 70, "valid config parses");
ok(valid.excludedTitleKeywords.length === 2, "exclusion lists parse");
ok(valid.preferences.picks.length === 1, "picks parse");

let threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: 101 });
} catch {
	threw = true;
}
ok(threw, "threshold > 100 rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: -1 });
} catch {
	threw = true;
}
ok(threw, "threshold < 0 rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, notifyThreshold: 70.5 });
} catch {
	threw = true;
}
ok(threw, "float threshold rejected");

threw = false;
try {
	scoringConfigSchema.parse({ ...base, excludedTitleKeywords: [1] });
} catch {
	threw = true;
}
ok(threw, "non-string exclusion entry rejected");

threw = false;
try {
	scoringConfigSchema.parse({
		...base,
		preferences: { picks: [{ optionId: "tech:go", stance: "nice" }] },
	});
} catch {
	threw = true;
}
ok(threw, "a pick missing source is rejected");

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
ok(threw, "a negative salary floor amount is rejected");

const withFloor = scoringConfigSchema.parse({
	...base,
	preferences: {
		...base.preferences,
		salaryFloor: { amount: 55000, currency: "GBP" },
	},
});
ok(
	withFloor.preferences.salaryFloor?.amount === 55000,
	"a salary floor parses",
);
