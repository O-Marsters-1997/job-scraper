import assert from "node:assert/strict";
import type { Draft, LayoutBlock } from "@/types/tailoring";
import type { DraftEditorState } from "./useDraftEditor";
import { createRolePicks } from "./useRolePicks";

const block = (text: string, slotId = "", bullet = false) =>
	({
		slotId,
		bullet: bullet ? { glyph: "-", level: 0, size: 10 } : null,
		runs: [{ text }],
	}) as unknown as LayoutBlock;

const withStack = (stack: boolean) => {
	const heading = block("Software Engineer | Spectinga | Remote");
	const stackLine = block("Golang, Postgres, AWS");
	const bullet = block("Built things", "s1", true);
	const blocks = [
		block("Experience"),
		heading,
		...(stack ? [stackLine] : []),
		bullet,
		block("Second bullet", "s2", true),
	];
	const draft = {
		provenance: {
			positions: [
				{ positionId: "p1", bullets: [{ slotId: "s1", achievements: [] }] },
			],
		},
		content: {
			positions: [
				{
					positionId: "p1",
					headingText: "Software Engineer | Spectinga | Remote",
					bullets: [],
				},
			],
		},
	} as unknown as Draft;
	const picks = createRolePicks({
		draft: () => draft,
		editor: { blocks: () => blocks } as unknown as DraftEditorState,
		bank: () => [],
		positions: () => [],
	});
	return { picks, heading, stackLine };
};

{
	const { picks, heading, stackLine } = withStack(true);
	assert.equal(picks.headings().get(heading), "p1");
	assert.equal(picks.headings().has(stackLine), false);
	assert.deepEqual(picks.roleSlots("p1"), ["s1", "s2"]);
}

{
	const { picks, heading } = withStack(false);
	assert.equal(picks.headings().get(heading), "p1");
	assert.deepEqual(picks.roleSlots("p1"), ["s1", "s2"]);
}
