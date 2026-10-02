import assert from "node:assert/strict";
import { createSaveLoop, type SaveStatus } from "./saveLoop";

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

function harness(fail = false) {
	const log: string[] = [];
	let running = 0;
	let maxRunning = 0;
	let sends = 0;
	let failNext = fail;
	const loop = createSaveLoop({
		delay: 10,
		send: async () => {
			sends++;
			running++;
			maxRunning = Math.max(maxRunning, running);
			await wait(20);
			running--;
			if (failNext) {
				failNext = false;
				throw new Error("down");
			}
		},
		onStatus: (s: SaveStatus) => log.push(s),
	});
	return { loop, log, sends: () => sends, maxRunning: () => maxRunning };
}

{
	const h = harness();
	h.loop.schedule();
	h.loop.schedule();
	h.loop.schedule();
	await h.loop.flush();
	assert.equal(h.sends(), 1, "a burst of edits is one save");
	assert.equal(h.log.at(-1), "saved");
	assert.equal(h.loop.pending(), false);
}

{
	const h = harness();
	h.loop.schedule();
	await wait(20);
	h.loop.schedule();
	h.loop.schedule();
	await wait(20);
	await h.loop.flush();
	assert.equal(h.sends(), 2, "edits during a save coalesce into one more");
	assert.equal(h.maxRunning(), 1, "saves never overlap");
}

{
	const h = harness(true);
	h.loop.schedule();
	await h.loop.flush();
	assert.equal(h.log.at(-1), "failed", "a failed save is reported");
	assert.equal(await h.loop.flush(), false, "flush reports the failed save");
	h.loop.retry();
	assert.equal(await h.loop.flush(), true, "flush reports a good save");
	assert.equal(h.log.at(-1), "saved", "retry saves the full state again");
}

{
	const h = harness();
	h.loop.schedule();
	assert.equal(h.loop.pending(), true, "a debounced save is pending");
	await h.loop.flush();
	assert.equal(h.sends(), 1, "flush sends without waiting out the delay");
}
