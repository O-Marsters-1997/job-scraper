import assert from "node:assert/strict";
import { pendingIds } from "./session";

assert.deepEqual(pendingIds(["a", "b", "c"], []), ["a", "b", "c"]);
assert.deepEqual(pendingIds(["a", "b", "c"], ["a"]), ["b", "c", "a"]);
assert.deepEqual(pendingIds(["b", "c"], ["a", "c"]), ["b", "c"]);
