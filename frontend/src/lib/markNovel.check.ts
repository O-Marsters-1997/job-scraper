import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import type { Segment } from "@/types/tailoring";
import { markNovel } from "./markNovel";

type Case = { name: string; text: string; sources: string[]; want: Segment[] };

const golden: Case[] = JSON.parse(
	readFileSync(
		new URL(
			"../../../internal/services/cvtailor/testdata/novel.json",
			import.meta.url,
		),
		"utf8",
	),
);

assert.ok(golden.length > 0, "the shared golden is not empty");
for (const c of golden) {
	assert.deepEqual(markNovel(c.text, c.sources), c.want, c.name);
}
