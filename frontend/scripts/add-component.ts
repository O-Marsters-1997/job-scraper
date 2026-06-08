#!/usr/bin/env bun
/**
 * Fetches a Kobalte UI component from the Zaidan registry and installs it.
 *
 * Usage:   bun run scripts/add-component.ts <name>
 * Example: bun run scripts/add-component.ts dialog
 *
 * Steps:
 *   1. Fetches src/registry/kobalte/ui/<name>.tsx from github.com/carere/zaidan
 *   2. Rewrites @/registry/kobalte/ui/ → @/components/ui/ import paths
 *   3. Extracts z-* CSS class names and injects their @apply definitions
 *      from style-luma.css into src/styles.css inside @layer components
 *
 * Browse available components:
 *   https://github.com/carere/zaidan/tree/main/src/registry/kobalte/ui
 */

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const ZAIDAN_UI =
	"https://raw.githubusercontent.com/carere/zaidan/main/src/registry/kobalte/ui";
const ZAIDAN_CSS_URL =
	"https://raw.githubusercontent.com/carere/zaidan/main/src/registry/kobalte/styles/style-luma.css";

const ROOT = fileURLToPath(new URL("..", import.meta.url));
const UI_DIR = join(ROOT, "src/components/ui");
const STYLES_PATH = join(ROOT, "src/styles.css");

const name = process.argv[2];
if (!name) {
	console.error("Usage: bun run scripts/add-component.ts <component-name>");
	process.exit(1);
}

async function get(url: string): Promise<string | null> {
	const res = await fetch(url);
	return res.ok ? res.text() : null;
}

// 1. Fetch component TSX
process.stdout.write(`Fetching ${name}… `);
const raw = await get(`${ZAIDAN_UI}/${name}.tsx`);
if (!raw) {
	console.error(
		`\n✗ Not found in Zaidan registry. Available components:\n` +
			`  https://github.com/carere/zaidan/tree/main/src/registry/kobalte/ui`,
	);
	process.exit(1);
}
console.log("ok");

// 2. Fix internal import paths
const tsx = raw.replaceAll("@/registry/kobalte/ui/", "@/components/ui/");

// 3. Write component file
mkdirSync(UI_DIR, { recursive: true });
const outPath = join(UI_DIR, `${name}.tsx`);
const existed = existsSync(outPath);
writeFileSync(outPath, tsx);
console.log(
	`✓ ${existed ? "overwrote" : "wrote"} src/components/ui/${name}.tsx`,
);

// 4. Extract z-* class names referenced in this component
const zClasses = [
	...new Set([...tsx.matchAll(/\bz-[\w-]+/g)].map((m) => m[0])),
];
if (zClasses.length === 0) {
	process.exit(0);
}
console.log(`  CSS classes needed: ${zClasses.join(", ")}`);

// 5. Fetch Zaidan style definitions
process.stdout.write("Fetching style-luma.css… ");
const cssSource = await get(ZAIDAN_CSS_URL);
if (!cssSource) {
	console.warn(
		"\n⚠  Could not fetch style-luma.css. Add z-* definitions to src/styles.css manually.",
	);
	process.exit(0);
}
console.log("ok");

// 6. Extract matching class blocks (single-level braces only — matches Zaidan's format)
const blocks: string[] = [];
for (const cls of zClasses) {
	const escaped = cls.replace(/[-]/g, "\\$&");
	const re = new RegExp(`\\.${escaped}\\s*\\{[\\s\\S]*?\\}`, "g");
	for (const m of cssSource.matchAll(re)) blocks.push(m[0].trim());
}

if (blocks.length === 0) {
	console.warn(
		"⚠  No CSS blocks matched in style-luma.css — add z-* definitions manually.",
	);
	process.exit(0);
}

// 7. Inject only missing class definitions into styles.css
const styles = readFileSync(STYLES_PATH, "utf8");
const missing = blocks.filter((block) => {
	const cls = block.match(/\.([\w-]+)/)?.[1];
	return cls && !styles.includes(`.${cls}`);
});

if (missing.length === 0) {
	console.log("  All CSS classes already present in styles.css.");
} else {
	const body = missing.map((b) => `  ${b.replace(/\n/g, "\n  ")}`).join("\n");
	const injection = `\n@layer components {\n  /* ${name} — zaidan/style-luma.css */\n${body}\n}\n`;
	writeFileSync(STYLES_PATH, styles + injection);
	console.log(`✓ injected ${missing.length} class(es) into src/styles.css`);
	console.log(
		"  ⚠  Review injected CSS: zaidan tokens (bg-popover, bg-input, bg-secondary, etc.)\n" +
			"     may differ from this project's @theme — adjust to match before committing.",
	);
}
