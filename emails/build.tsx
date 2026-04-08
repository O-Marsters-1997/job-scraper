import { render } from "@react-email/render";
import { rm, mkdir, writeFile } from "fs/promises";
import { join, dirname } from "path";
import { fileURLToPath } from "url";
import DigestEmail from "./templates/digest";
import IndividualEmail from "./templates/individual";

const __dirname = dirname(fileURLToPath(import.meta.url));
const outDir = join(__dirname, "../internal/notify/templates");

// Clear and recreate the output directory so stale templates are never left behind.
await rm(outDir, { recursive: true, force: true });
await mkdir(outDir, { recursive: true });

const templates = [
  { name: "digest.tmpl", element: <DigestEmail /> },
  { name: "individual.tmpl", element: <IndividualEmail /> },
];

for (const { name, element } of templates) {
  const html = await render(element, { pretty: true });
  await writeFile(join(outDir, name), html, "utf-8");
  console.log(`✓ ${name}`);
}

console.log(`\nBuilt ${templates.length} template(s) → ${outDir}`);
