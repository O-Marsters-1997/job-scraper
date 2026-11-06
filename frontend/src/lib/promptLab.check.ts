import assert from "node:assert/strict";
import type { PromptLab } from "@/types/promptLab";
import {
	missingReferences,
	partTexts,
	rawMessage,
	sentRows,
} from "./promptLab";

const lab: PromptLab = {
	generation: "profile-redraft",
	main: "a.md",
	parts: [
		{ path: "a.md", text: "Use [b.md](b.md), job.md and gone.md." },
		{ path: "b.md", text: "b" },
	],
	documents: [{ path: "job.md", text: "job" }],
	checks: [],
};

assert.deepEqual(missingReferences(lab), ["gone.md"]);
assert.deepEqual(
	sentRows(lab).map((r) => r.path),
	["job.md", "b.md", "a.md"],
);
assert.equal(
	rawMessage(lab),
	'<documents>\n<document path="job.md">\njob\n</document>\n<document path="b.md">\nb\n</document>\n</documents>\n\n<instructions path="a.md">\nUse [b.md](b.md), job.md and gone.md.\n</instructions>',
);
assert.deepEqual(partTexts(lab, { "b.md": "B" }), {
	"a.md": lab.parts[0]?.text,
	"b.md": "B",
});
console.log("✓ promptLab checks passed");
