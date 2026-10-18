import assert from "node:assert/strict";
import { Window } from "happy-dom";
import { findQuote, quoteFromRange } from "./noteAnchor";

function page(html: string) {
	const window = new Window();
	const doc = window.document;
	doc.body.innerHTML = html;
	return {
		doc: doc as unknown as Document,
		root: doc.body as unknown as Node,
	};
}

function select(doc: Document, from: Node, a: number, to: Node, b: number) {
	const r = doc.createRange();
	r.setStart(from, a);
	r.setEnd(to, b);
	return r;
}

{
	const { doc, root } = page(
		"<p>We use Go and Rust.</p><p>Remote first, Go always.</p>",
	);
	const [p1, p2] = Array.from(doc.querySelectorAll("p"));
	const q = quoteFromRange(
		root,
		select(doc, p1?.firstChild as Node, 7, p2?.firstChild as Node, 6),
	);
	assert.equal(q.exact, "Go and Rust.Remote");
	assert.equal(q.prefix, "We use ");
	assert.equal(q.suffix, " first, Go always.");
	assert.equal(findQuote(root, q)?.toString(), q.exact);
}

{
	const long = "x".repeat(100);
	const { doc, root } = page(`<p>${long}needle${long}</p>`);
	const t = doc.querySelector("p")?.firstChild as Node;
	const q = quoteFromRange(root, select(doc, t, 100, t, 106));
	assert.equal(q.prefix.length, 30);
	assert.equal(q.suffix.length, 30);
}

{
	const { doc, root } = page(
		"<p>apply now</p><p>or apply later</p><p>apply today</p>",
	);
	const t = doc.querySelectorAll("p")[1]?.firstChild as Node;
	const q = quoteFromRange(root, select(doc, t, 3, t, 8));
	assert.equal(q.exact, "apply");
	const found = findQuote(root, q);
	assert.equal(found?.startContainer, t);
	assert.equal(found?.startOffset, 3);
}

{
	const { root } = page("<p>nothing relevant</p>");
	assert.equal(
		findQuote(root, { exact: "gone", prefix: "", suffix: "" }),
		null,
	);
}

{
	const { doc, root } = page(`<p>${"a".repeat(2500)}END</p>`);
	const t = doc.querySelector("p")?.firstChild as Node;
	const q = quoteFromRange(root, select(doc, t, 0, t, 2500));
	assert.equal(q.exact.length, 2000);
	assert.equal(q.suffix, "a".repeat(30));
}

console.log("✓ noteAnchor checks passed");
