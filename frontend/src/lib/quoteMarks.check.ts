import assert from "node:assert/strict";
import { findQuoteSpans } from "./quoteMarks";

const text = "We need a Go engineer.\nWants   a mentor (really).";

assert.deepEqual(
	findQuoteSpans(text, [
		{ key: "b", quote: "Wants a mentor (really)" },
		{ key: "a", quote: "Go engineer" },
	]),
	[
		{ start: 10, end: 21, key: "a" },
		{ start: 23, end: 48, key: "b" },
	],
);

assert.deepEqual(
	findQuoteSpans(text, [
		{ key: "a", quote: "Go engineer" },
		{ key: "b", quote: "a Go" },
		{ key: "c", quote: "" },
		{ key: "d", quote: "missing" },
	]),
	[{ start: 8, end: 12, key: "b" }],
);
