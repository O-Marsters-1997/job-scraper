import assert from "node:assert/strict";
import { scoringOptionsSchema } from "../types/scoringOptions";

const base = {
	dimensions: [{ key: "tech", kind: "pair", stances: ["nice", "avoid"] }],
	options: [{ id: "tech:go", dimension: "tech", label: "Go" }],
};

const valid = scoringOptionsSchema.parse(base);
assert.ok(valid.dimensions.length === 1, "dimensions parse");
assert.ok(valid.options[0]?.id === "tech:go", "options parse");

let threw = false;
try {
	scoringOptionsSchema.parse({
		...base,
		dimensions: [{ ...base.dimensions[0], key: "nonsense" }],
	});
} catch {
	threw = true;
}
assert.ok(threw, "unknown dimension key rejected");

threw = false;
try {
	scoringOptionsSchema.parse({
		...base,
		options: [{ ...base.options[0], question: "leaked" }],
	});
} catch {
	threw = true;
}
assert.ok(!threw, "extra fields on an option are ignored, not rejected");
