import assert from "node:assert/strict";
import type { DraftLayout } from "@/types/tailoring";
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

const run = (font: string) => ({
	text: "x",
	font,
	size: 11,
	bold: false,
	italic: false,
	underline: false,
	color: "",
	link: "",
});
const layout = {
	page: {
		width: 1,
		height: 1,
		marginTop: 0,
		marginBottom: 0,
		marginLeft: 0,
		marginRight: 0,
	},
	warnings: [],
	blocks: [{ runs: [run("Calibri"), run("Roboto"), run("Roboto")] }],
} as unknown as DraftLayout;
assert.deepEqual(
	fontWarnings(layout),
	["Roboto is not installed here, so line breaks may differ."],
	"each unknown font warns once",
);
