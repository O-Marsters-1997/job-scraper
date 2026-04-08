import { render } from "@react-email/render";
import { rm, mkdir, writeFile } from "fs/promises";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import DigestEmail from "./templates/digest";
import IndividualEmail from "./templates/individual";
import { JobCard } from "./components/job-card";

const __dirname = dirname(fileURLToPath(import.meta.url));
const outDir = join(__dirname, "../internal/notify/templates");

// Placeholder values that get replaced with Go template directives in the HTML output.
const placeholders = {
  title: "%%GO_TITLE%%",
  company: "%%GO_COMPANY%%",
  location: "%%GO_LOCATION%%",
  url: "%%GO_URL%%",
  remuneration: "%%GO_REMUNERATION%%",
  rangeStart: "%%GO_RANGE_START%%",
  rangeEnd: "%%GO_RANGE_END%%",
};

const directiveMap: Record<string, string> = {
  [placeholders.title]: "{{.Title}}",
  [placeholders.company]: "{{.Company}}",
  [placeholders.location]: "{{.Location}}",
  [placeholders.url]: "{{.URL}}",
  [placeholders.remuneration]: "{{.Remuneration}}",
  [placeholders.rangeStart]: "{{range .Jobs}}",
  [placeholders.rangeEnd]: "{{end}}",
};

function injectGoDirectives(html: string): string {
  let result = html;

  // Replace all placeholders with Go directives.
  for (const [placeholder, directive] of Object.entries(directiveMap)) {
    result = result.replaceAll(placeholder, directive);
  }

  // Wrap the remuneration <p> element with {{if .Remuneration}}...{{end}}.
  result = result.replace(
    /(\s*)(<p[^>]*>)\s*\{\{\.Remuneration\}\}\s*(<\/p>)/,
    "$1{{if .Remuneration}}$2{{.Remuneration}}$3{{end}}",
  );

  return result;
}

await rm(outDir, { recursive: true, force: true });
await mkdir(outDir, { recursive: true });

const goJob = {
  title: placeholders.title,
  company: placeholders.company,
  location: placeholders.location,
  url: placeholders.url,
  remuneration: placeholders.remuneration,
};

const templates = [
  {
    name: "digest.tmpl",
    element: (
      <DigestEmail>
        {placeholders.rangeStart}
        <JobCard {...goJob} />
        {placeholders.rangeEnd}
      </DigestEmail>
    ),
  },
  {
    name: "individual.tmpl",
    element: <IndividualEmail {...goJob} />,
  },
];

for (const { name, element } of templates) {
  const html = await render(element, { pretty: true });
  const tmpl = injectGoDirectives(html);
  await writeFile(join(outDir, name), tmpl, "utf-8");
  console.log(`✓ ${name}`);
}

console.log(`\nBuilt ${templates.length} template(s) → ${outDir}`);
