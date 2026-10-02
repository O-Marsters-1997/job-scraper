import assert from "node:assert/strict";
import { resolveFont } from "./docFonts";

assert.match(resolveFont("Calibri").css, /^Carlito/, "Calibri maps to Carlito");
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
assert.match(
	resolveFont("Open Sans Serif").css,
	/^Arimo/,
	"a sans name wins over a serif hint",
);
