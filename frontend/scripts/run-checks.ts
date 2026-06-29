// Finds and runs every src/**/*.check.ts self-check file.
// Each file asserts at import time and throws on failure.
// Run from frontend/: bun run scripts/run-checks.ts
import { Glob } from "bun";

const glob = new Glob("src/**/*.check.ts");
const files: string[] = [];
for await (const file of glob.scan(".")) {
	files.push(file);
}

for (const file of files.sort()) {
	await import(`../${file}`);
}
