import { render } from "@react-email/render";
import { mkdir, writeFile } from "fs/promises";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import IndividualEmail from "./templates/individual";

const __dirname = dirname(fileURLToPath(import.meta.url));
const outDir = join(__dirname, "../internal/services/notify/templates");

const placeholders = {
	title: "%%GO_TITLE%%",
	company: "%%GO_COMPANY%%",
	location: "%%GO_LOCATION%%",
	url: "%%GO_URL%%",
	remuneration: "%%GO_REMUNERATION%%",
};

const directiveMap: Record<string, string> = {
	[placeholders.title]: "{{.Title}}",
	[placeholders.company]: "{{.Company}}",
	[placeholders.location]: "{{.Location}}",
	[placeholders.url]: "{{.URL}}",
	[placeholders.remuneration]: "{{.Remuneration}}",
};

function injectGoDirectives(html: string): string {
	let result = html;

	for (const [placeholder, directive] of Object.entries(directiveMap)) {
		result = result.replaceAll(placeholder, directive);
	}

	const remuneration = /(\s*)(<p[^>]*>)\s*\{\{\.Remuneration\}\}\s*(<\/p>)/;
	if (!remuneration.test(result))
		throw new Error("Remuneration paragraph missing");
	result = result.replace(
		remuneration,
		"$1{{if .Remuneration}}$2{{.Remuneration}}$3{{end}}",
	);

	return result;
}

const goJob = {
	title: placeholders.title,
	company: placeholders.company,
	location: placeholders.location,
	url: placeholders.url,
	remuneration: placeholders.remuneration,
};

const html = injectGoDirectives(
	await render(<IndividualEmail {...goJob} />, { pretty: true }),
);
await mkdir(outDir, { recursive: true });
await writeFile(join(outDir, "individual.tmpl"), html, "utf-8");
console.log(`Built individual.tmpl → ${outDir}`);
