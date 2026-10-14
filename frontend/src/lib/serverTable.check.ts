import assert from "node:assert/strict";
import {
	clampPage,
	filterSignature,
	nextSelection,
	pageCount,
} from "./serverTable";

assert.equal(pageCount(0, 25), 1);
assert.equal(pageCount(25, 25), 1);
assert.equal(pageCount(26, 25), 2);

assert.equal(clampPage(3, 60, 25), 3);
assert.equal(clampPage(3, 50, 25), 2);
assert.equal(clampPage(4, 0, 25), 1);

assert.equal(
	filterSignature({ q: "a", page: 2 }),
	filterSignature({ q: "a", page: 5 }),
	"page is not part of the filter signature",
);
assert.notEqual(
	filterSignature({ q: "a", page: 2 }),
	filterSignature({ q: "b", page: 2 }),
);

console.log("✓ serverTable checks passed");

{
	const rowId = (item: { id: string }) => item.id;
	const page1 = [{ id: "a" }, { id: "b" }];
	const page2 = [{ id: "c" }];
	const onPage1 = nextSelection({}, { a: true }, page1, rowId);
	assert.deepEqual(Object.keys(onPage1), ["a"]);

	const onPage2 = nextSelection(onPage1, { a: true, c: true }, page2, rowId);
	assert.deepEqual(
		Object.keys(onPage2),
		["a", "c"],
		"a selection made on page 1 survives selecting on page 2",
	);
	assert.equal(onPage2.a, page1[0], "earlier rows keep their loaded item");

	assert.deepEqual(
		Object.keys(nextSelection(onPage2, { a: false, c: true }, page2, rowId)),
		["c"],
	);
}
