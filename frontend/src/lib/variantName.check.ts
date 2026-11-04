import assert from "node:assert/strict";
import { variantName } from "./variantName";

assert.equal(
	variantName({ text: "Cut latency", label: " Checkout " }),
	"Checkout",
);
assert.equal(
	variantName({ text: "Cut latency by half", label: "" }),
	"Cut latency by half",
);
assert.equal(
	variantName({ text: "one two three four five six seven eight", label: "" }),
	"one two three four five six…",
);
