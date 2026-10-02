import assert from "node:assert/strict";
import { resolveFont } from "./docFonts";

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
