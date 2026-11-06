import assert from "node:assert/strict";
import { parseReply, tailoringPrompt } from "./promptHandoff";

const texts = {
	"main.md": "Write it.",
	"voice.md": "Be plain.",
	"job.md": "A job",
};

const reply = `Sure.
<file path="voice.md">
Be warm.
</file>
<file path="job.md">
edited
</file>
<file path="nope.md">x</file>
<file path="main.md">Write it.  </file>
Voice: warmer.`;
const parsed = parseReply(reply, texts);
assert.deepEqual(parsed.changes, [
	{ path: "voice.md", from: "Be plain.", to: "Be warm." },
	{ path: "job.md", from: "A job", to: "edited" },
]);
assert.deepEqual(parsed.skipped, ["nope.md"]);
assert.equal(parsed.empty, false);
assert.equal(parseReply("no blocks", texts).empty, true);

const base = {
	streaming: false,
	findings: [],
	lines: 2,
	steer: "",
	parts: {},
	cost: 0,
};
const prompt = tailoringPrompt({
	brief: "make it punchier",
	lab: {
		generation: "g1",
		main: "main.md",
		parts: [
			{ path: "main.md", text: "Write it.", source: "default" },
			{ path: "voice.md", text: "Be plain.", source: "default" },
		],
		documents: [{ path: "job.md", text: "A job" }],
		checks: ["Length: 80 words"],
	},
	texts,
	tries: [
		{ ...base, n: 1, text: "first", changed: [] },
		{ ...base, n: 2, text: "second", changed: ["voice.md"], steer: "shorter" },
	],
});
for (const s of [
	"make it punchier",
	"Write it.",
	"Be plain.",
	"A job",
	"Length: 80 words",
	"after editing voice.md",
	'steer "shorter"',
])
	assert.ok(prompt.includes(s), s);
assert.ok(prompt.indexOf("### Try 2") < prompt.indexOf("### Try 1"));
assert.ok(!prompt.includes("SYSTEM PROMPT SECRET"));
