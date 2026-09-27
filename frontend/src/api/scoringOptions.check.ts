import { scoringOptionsSchema } from "../types/scoringOptions";

function ok(cond: boolean, msg: string) {
	if (!cond) throw new Error(`FAIL: ${msg}`);
}

const base = {
	dimensions: [{ key: "tech", kind: "pair", stances: ["nice", "avoid"] }],
	options: [{ id: "tech:go", dimension: "tech", label: "Go" }],
};

const valid = scoringOptionsSchema.parse(base);
ok(valid.dimensions.length === 1, "dimensions parse");
ok(valid.options[0]?.id === "tech:go", "options parse");

let threw = false;
try {
	scoringOptionsSchema.parse({
		...base,
		dimensions: [{ ...base.dimensions[0], key: "nonsense" }],
	});
} catch {
	threw = true;
}
ok(threw, "unknown dimension key rejected");

threw = false;
try {
	scoringOptionsSchema.parse({
		...base,
		options: [{ ...base.options[0], question: "leaked" }],
	});
} catch {
	threw = true;
}
ok(!threw, "extra fields on an option are ignored, not rejected");
