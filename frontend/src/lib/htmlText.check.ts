import assert from "node:assert/strict";
import { clipSpans, htmlToParts, joinParts } from "./htmlText";
import { findQuoteSpans } from "./quoteMarks";

const parts = htmlToParts(
	"The Role\n\n    We build things that\n    scale well.\n\n- Go\n- SQL\n",
);
assert.deepEqual(parts, [
	{ kind: "p", text: "The Role" },
	{ kind: "p", text: "We build things that scale well." },
	{ kind: "li", text: "Go" },
	{ kind: "li", text: "SQL" },
]);

const text = joinParts(parts);
const spans = findQuoteSpans(text, [
	{ key: "a", quote: "scale well" },
	{ key: "b", quote: "Go SQL" },
]);
assert.deepEqual(
	clipSpans(spans, parts).map((c) => c.spans),
	[
		[],
		[{ key: "a", start: 21, end: 31 }],
		[{ key: "b", start: 0, end: 2 }],
		[{ key: "b", start: 0, end: 3 }],
	],
);
