import { scoringConfigSchema } from "./scoringConfig";

function ok(cond: boolean, msg: string) {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}

// valid config passes
const valid = scoringConfigSchema.parse({
	suitabilityRubric:
		"I am a Go engineer looking for distributed systems roles.",
	relevanceCutoff: 40,
	notifyThreshold: 70,
});
ok(valid.relevanceCutoff === 40, "valid config parses");

// cutoff above 100 is rejected
let threw = false;
try {
	scoringConfigSchema.parse({
		suitabilityRubric: "",
		relevanceCutoff: 101,
		notifyThreshold: 70,
	});
} catch {
	threw = true;
}
ok(threw, "cutoff > 100 rejected");

// threshold below 0 is rejected
threw = false;
try {
	scoringConfigSchema.parse({
		suitabilityRubric: "",
		relevanceCutoff: 50,
		notifyThreshold: -1,
	});
} catch {
	threw = true;
}
ok(threw, "threshold < 0 rejected");

// non-integer is rejected
threw = false;
try {
	scoringConfigSchema.parse({
		suitabilityRubric: "",
		relevanceCutoff: 50.5,
		notifyThreshold: 70,
	});
} catch {
	threw = true;
}
ok(threw, "float cutoff rejected");
