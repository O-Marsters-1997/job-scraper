import { Glob } from "bun";

const glob = new Glob("src/**/*.check.ts");
const files: string[] = [];
for await (const file of glob.scan(".")) {
	files.push(file);
}

for (const file of files.sort()) {
	await import(`../${file}`);
}
