import assert from "node:assert/strict";
import { type CardSlot, cardTops } from "./cardLayout";

const card = (key: string, desired: number, height = 40): CardSlot => ({
	key,
	desired,
	height,
});
const GAP = 10;

const apart = [card("a", 0), card("b", 100), card("c", 200)];
assert.deepEqual(
	cardTops(apart, 1, GAP),
	{ a: 0, b: 100, c: 200 },
	"cards with room stay level with their lines",
);

const crowded = [
	card("a", 100),
	card("b", 110),
	card("c", 120),
	card("d", 130),
];
for (let pinned = 0; pinned < crowded.length; pinned++) {
	const tops = cardTops(crowded, pinned, GAP);
	const pin = crowded[pinned] as CardSlot;
	assert.equal(
		tops[pin.key],
		pin.desired,
		"the pinned card sits at its anchor",
	);
	for (let i = 1; i < crowded.length; i++) {
		const prev = crowded[i - 1] as CardSlot;
		const cur = crowded[i] as CardSlot;
		assert.ok(
			(tops[cur.key] ?? 0) >= (tops[prev.key] ?? 0) + prev.height + GAP,
			`${cur.key} clears ${prev.key} with pinned=${pinned}`,
		);
	}
}

assert.deepEqual(
	cardTops(crowded, 1, GAP),
	{ a: 60, b: 110, c: 160, d: 210 },
	"cards above are pushed up and cards below pushed down",
);
assert.deepEqual(cardTops([], 0, GAP), {}, "no cards, no tops");
assert.equal(
	cardTops([card("a", 5)], -1, GAP).a,
	5,
	"a missing pin falls back to the first card",
);
