import assert from "node:assert/strict";
import type { DraftLayout, LayoutBlock, LayoutRun } from "@/types/tailoring";
import { fontWarnings, resolveFont } from "./docFonts";

assert.equal(resolveFont("Calibri").known, true, "Calibri maps to Carlito");
assert.match(
	resolveFont(" calibri ").css,
	/^Carlito/,
	"names ignore case and padding",
);
assert.equal(resolveFont("Cambria").ratio, 1.15, "Cambria maps to Caladea");
assert.match(
	resolveFont("Georgia").css,
	/^Tinos/,
	"an unknown serif falls back to Tinos",
);
assert.match(
	resolveFont("Roboto").css,
	/^Arimo/,
	"an unknown sans falls back to Arimo",
);
assert.equal(resolveFont("Georgia").known, false, "a fallback is flagged");

const run = (font: string): LayoutRun => ({
	text: "x",
	font,
	size: 11,
	bold: false,
	italic: false,
	underline: false,
	color: "",
	link: "",
});
const block = (runs: LayoutRun[]): LayoutBlock => ({
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
	runs,
});
const layout: DraftLayout = {
	page: {
		width: 1,
		height: 1,
		marginTop: 0,
		marginBottom: 0,
		marginLeft: 0,
		marginRight: 0,
	},
	warnings: [],
	blocks: [block([run("Calibri"), run("Roboto"), run("Roboto")])],
};
assert.deepEqual(
	fontWarnings(layout),
	["Roboto is not installed here, so line breaks may differ."],
	"each unknown font warns once",
);
