import assert from "node:assert/strict";
import { Window } from "happy-dom";
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

const win = new Window() as unknown as {
	DOMParser: unknown;
	Node: unknown;
};
Object.assign(globalThis, { DOMParser: win.DOMParser, Node: win.Node });
const linkedin = htmlToParts(
	"<p><strong>About BJAK</strong></p><p>We build <strong>fast</strong> things.</p>" +
		"<strong>The Role</strong><br>Own the API.<br><ul><li>Go</li></ul>" +
		"Seniority level: Senior Employment type: Full-time Industries: Software",
);
assert.deepEqual(linkedin, [
	{ kind: "h", text: "About BJAK" },
	{ kind: "p", text: "We build fast things." },
	{ kind: "h", text: "The Role" },
	{ kind: "p", text: "Own the API." },
	{ kind: "li", text: "Go" },
	{ kind: "p", text: "Seniority level: Senior" },
	{ kind: "p", text: "Employment type: Full-time" },
	{ kind: "p", text: "Industries: Software" },
]);
