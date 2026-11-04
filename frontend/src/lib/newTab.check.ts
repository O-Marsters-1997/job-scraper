import assert from "node:assert/strict";
import { openInNewTab } from "./newTab";

function fakeWindow(blocked = false) {
	const tab = {
		location: { href: "" },
		closed: false,
		close() {
			this.closed = true;
		},
	};
	const assigned: string[] = [];
	const opened: string[] = [];
	Object.assign(globalThis, {
		window: {
			open: (url: string) => {
				opened.push(url);
				return blocked ? null : tab;
			},
			location: { assign: (url: string) => assigned.push(url) },
		},
	});
	return { tab, assigned, opened };
}

{
	const w = fakeWindow();
	let openedBeforeAwait = false;
	await openInNewTab(async () => {
		openedBeforeAwait = w.opened.length === 1;
		return "/tailoring/drafts/d1";
	});
	assert.ok(openedBeforeAwait);
	assert.equal(w.tab.location.href, "/tailoring/drafts/d1");
	assert.equal(w.tab.closed, false);
}

{
	const w = fakeWindow();
	await assert.rejects(
		openInNewTab(async () => {
			throw new Error("create failed");
		}),
		/create failed/,
	);
	assert.equal(w.tab.closed, true);
}

{
	const w = fakeWindow(true);
	await openInNewTab(async () => "/tailoring/drafts/d2");
	assert.deepEqual(w.assigned, ["/tailoring/drafts/d2"]);
}

Reflect.deleteProperty(globalThis, "window");
