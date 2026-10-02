import assert from "node:assert/strict";
import { createSseParser } from "./sse";

const seen: string[] = [];
const feed = createSseParser((event, data) => seen.push(`${event}:${data}`));

feed('event: delta\ndata: {"text":"Cu');
assert.deepEqual(seen, []);
feed('t"}\n\nevent: delta\r\ndata: {"text":" p99"}\r\n\r\n');
feed('event: done\ndata: {"text":"Cut p99","findings":[]}\n\n');
feed(": keep-alive\n\n");
assert.deepEqual(seen, [
	'delta:{"text":"Cut"}',
	'delta:{"text":" p99"}',
	'done:{"text":"Cut p99","findings":[]}',
]);
