import assert from "node:assert/strict";
import { oklchLightness, oklchToHex } from "./color";

assert.equal(
	oklchToHex("oklch(55% .18 285)"),
	oklchToHex("oklch(0.55 0.18 285)"),
);
assert.equal(oklchLightness("oklch(55% .18 285)"), 0.55);
assert.match(oklchToHex("oklch(55% .18 285)"), /^#[0-9a-f]{6}$/);
assert.equal(
	oklchToHex("oklch(55% .18 285 / 50%)"),
	oklchToHex("oklch(0.55 0.18 285 / 0.5)"),
);
assert.match(oklchToHex("oklch(55% .18 285 / .5)"), /^#[0-9a-f]{6}80$/);
assert.equal(oklchToHex("#123456"), "#123456");
