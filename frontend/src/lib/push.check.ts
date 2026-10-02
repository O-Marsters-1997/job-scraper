import assert from "node:assert/strict";
import { urlBase64ToBytes } from "./push";

assert.deepEqual([...urlBase64ToBytes("-_8")], [251, 255]);
assert.deepEqual([...urlBase64ToBytes("AQID")], [1, 2, 3]);

console.log("✓ push checks passed");
