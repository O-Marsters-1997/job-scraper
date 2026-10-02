import assert from "node:assert/strict";
import type { LayoutBlock, LayoutBorder, LayoutRun } from "@/types/tailoring";
import {
	blockText,
	borderDraws,
	charsToSave,
	hasEndTab,
	isSpacer,
	linePt,
	maxCharsForFewerLines,
	pageFit,
	splitAtTab,
	trimParagraphEnd,
} from "./docLayout";

const run = (text: string, bold = false): LayoutRun => ({
	text,
	font: "Calibri",
	size: 11,
	bold,
	italic: false,
	underline: false,
	color: "",
	link: "",
});
const rule: LayoutBorder = {
	width: 0.75,
	color: "#000",
	padding: 1,
	dash: "solid",
};
const block = (over: Partial<LayoutBlock>): LayoutBlock => ({
	slotId: "",
	section: "",
	align: "left",
	lineSpacing: 100,
	spaceAbove: 0,
	spaceBelow: 0,
	indentStart: 0,
	indentFirstLine: 0,
	borderTop: null,
	borderBottom: null,
	tabStops: [],
	bullet: null,
	runs: [run("x\n")],
	...over,
});

assert.equal(
	linePt(11, 1.2207, 115),
	11 * 1.2207 * 1.15,
	"line box is size x ratio x spacing",
);
assert.equal(linePt(10, 1.2, 0), 12, "an unset spacing is 100%");

assert.deepEqual(
	borderDraws([
		block({ borderBottom: rule }),
		block({ borderBottom: rule }),
		block({}),
	]),
	[
		{ top: true, bottom: false },
		{ top: true, bottom: true },
		{ top: true, bottom: true },
	],
	"adjacent identical rules draw once, at the last",
);
assert.equal(
	borderDraws([
		block({ borderBottom: rule }),
		block({ borderBottom: { ...rule, width: 2 } }),
	])[0]?.bottom,
	true,
	"different rules both draw",
);

const [left, right] = splitAtTab([run("Acme\t"), run("  2020 - 2022", true)]);
assert.deepEqual(
	[left.map((r) => r.text), right.map((r) => `${r.text}|${r.bold}`)],
	[["Acme"], ["  2020 - 2022|true"]],
	"runs split at the tab and keep their style",
);
assert.equal(
	splitAtTab([run("a\t  b")])[1][0]?.text,
	"b",
	"text after the tab loses leading space",
);
assert.equal(
	hasEndTab(block({ tabStops: [{ offset: 500, alignment: "end" }] })),
	true,
);
assert.equal(
	hasEndTab(block({ tabStops: [{ offset: 36, alignment: "start" }] })),
	false,
);

assert.equal(
	isSpacer(block({ runs: [run("\n")] })),
	true,
	"a lone newline is a spacer",
);
assert.equal(isSpacer(block({})), false);
assert.equal(
	trimParagraphEnd([run("hi\n")])[0]?.text,
	"hi",
	"the paragraph mark is dropped",
);

assert.equal(
	charsToSave("x".repeat(110), 2, 0.1),
	10,
	"a sparse last line names its chars",
);
assert.equal(
	charsToSave("x".repeat(110), 2, 0.5),
	0,
	"a fuller last line gives no hint",
);
assert.equal(charsToSave("x", 1, 0.1), 0, "one line gives no hint");

assert.equal(
	blockText(block({ runs: [run("Led "), run("the team\n", true)] })),
	"Led the team",
	"block text joins the runs and drops the paragraph mark",
);

assert.deepEqual(pageFit(700, 720, 15), { over: false, lines: 1 });
assert.deepEqual(pageFit(720, 720, 15), { over: false, lines: 0 });
assert.deepEqual(pageFit(750, 720, 15), { over: true, lines: 2 });

assert.equal(
	maxCharsForFewerLines("x".repeat(250), 3, 0.5),
	Math.floor(100 * 2 * 0.97),
	"drops the last line, with a little slack",
);
assert.equal(
	maxCharsForFewerLines("x".repeat(40), 1, 0.4),
	40,
	"one line stays",
);
