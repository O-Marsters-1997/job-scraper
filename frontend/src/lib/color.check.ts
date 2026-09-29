import { hexAlpha } from "./color";

function eq(a: unknown, b: unknown): void {
	if (a !== b) throw new Error(`expected ${String(a)} to equal ${String(b)}`);
}

eq(hexAlpha("#6645d9", "26"), "#6645d926");
eq(hexAlpha("#ABC", "26"), "#aabbcc26");
eq(hexAlpha("#6645d9ff", "26"), "#6645d926");
eq(hexAlpha("var(--color-status-saved)", "26"), "var(--color-status-saved)");

console.log("✓ color checks passed");
