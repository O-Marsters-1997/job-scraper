import assert from "node:assert/strict";
import { listDiff, wordDiff } from "./wordDiff";

const marks = (a: string, b: string) =>
	wordDiff(a, b)
		.map((d) => `${d.op}:${d.text}`)
		.join("|");

assert.equal(marks("a b", "a b"), "same:a b", "equal text is one same run");
assert.equal(
	marks("cut latency", "cut p99 latency"),
	"same:cut |add:p99 |same:latency",
	"an inserted word is an add",
);
assert.equal(
	marks("cut p99 latency", "cut latency"),
	"same:cut |del:p99 |same:latency",
	"a dropped word is a del",
);
assert.equal(
	marks("built apis", "wrote apis"),
	"del:built|add:wrote|same: apis",
	"a swapped word is a del then an add",
);
assert.equal(marks("", "x"), "add:x", "an empty base is all adds");

assert.equal(
	listDiff(["Go", "SQL"], ["SQL", "Rust"])
		.map((d) => `${d.op}:${d.text}`)
		.join("|"),
	"del:Go|same:SQL|add:Rust",
	"skills diff as a list",
);
