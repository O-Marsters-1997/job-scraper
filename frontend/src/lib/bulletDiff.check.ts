import assert from "node:assert/strict";
import { bulletDiff } from "./bulletDiff";

const kinds = (base: string[], current: string[]) =>
	bulletDiff(base, current)
		.map((r) => `${r.kind}:${r.text}`)
		.join("|");

assert.equal(kinds(["a b", "c d"], ["a b", "c d"]), "same:a b|same:c d");
assert.equal(
	kinds(
		["ran on call", "wrote docs"],
		["ran on call", "wrote docs", "led migration"],
	),
	"same:ran on call|same:wrote docs|added:led migration",
	"a new unrelated bullet is added",
);
assert.equal(
	kinds(["ran on call", "wrote docs"], ["wrote docs"]),
	"same:wrote docs|removed:ran on call",
	"a dropped bullet is removed and the rest stay put",
);
assert.equal(
	kinds(
		["one two", "three four", "five six"],
		["five six", "one two", "three four"],
	),
	"moved:five six|same:one two|same:three four",
	"a reordered bullet is moved",
);
const rewritten = bulletDiff(
	["built and maintained public apis"],
	["built and scaled public apis for fintech"],
)[0];
assert.ok(
	rewritten?.kind === "rewritten" &&
		rewritten.from === "built and maintained public apis",
	"a similar bullet is a rewrite carrying its original",
);
assert.equal(
	kinds(["built public apis"], ["negotiated vendor contracts"]),
	"added:negotiated vendor contracts|removed:built public apis",
	"an unrelated bullet is an add plus a remove, not a rewrite",
);
