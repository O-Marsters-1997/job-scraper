import { scoringConfigSchema } from "./scoringConfig";

function ok(cond: boolean, msg: string) {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}

const base = {
	notifyThreshold: 70,
	excludedTitleKeywords: ["java", "sales"],
	excludedCompanies: ["acme corp"],
	excludedSeniority: ["intern", "junior"],
	excludedLocations: ["united states"],
	scoringQuestions: {
		profile: "I am a Go engineer looking for distributed systems roles.",
		criteria: [
			{
				key: "go_backend",
				instructions: "Does the job involve Go backend work?",
				true: "Go is a primary language",
				false: "Go isn't used",
				required: true,
			},
		],
		scale: ["Not relevant", "Weak", "Possible", "Strong", "Apply today"],
	},
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

const valid2 = scoringConfigSchema.parse(base);
ok(valid2.scoringQuestions.criteria.length === 1, "criteria parse");
ok(valid2.scoringQuestions.scale.length === 5, "scale parses");

threw = false;
try {
	scoringConfigSchema.parse({
		...base,
		scoringQuestions: {
			...base.scoringQuestions,
			criteria: [{ ...base.scoringQuestions.criteria[0], required: "yes" }],
		},
	});
} catch {
	threw = true;
}
ok(threw, "non-boolean criterion required rejected");
