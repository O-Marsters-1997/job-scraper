import { scoringConfigSchema } from "./scoringConfig";

function ok(cond: boolean, msg: string) {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}

const base = {
	suitabilityRubric:
		"I am a Go engineer looking for distributed systems roles.",
	notifyThreshold: 70,
	excludedTitleKeywords: ["java", "sales"],
	excludedCompanies: ["acme corp"],
	excludedSeniority: ["intern", "junior"],
	excludedLocations: ["united states"],
};

const valid = scoringConfigSchema.parse(base);
ok(valid.notifyThreshold === 70, "valid config parses");
ok(valid.excludedTitleKeywords.length === 2, "exclusion lists parse");

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
