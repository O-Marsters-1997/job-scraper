import assert from "node:assert/strict";
import { capitalise, uniqueCapitalised } from "./capitalise";

assert.deepEqual(capitalise("  acme corp "), "Acme Corp");
assert.deepEqual(capitalise("openAI"), "OpenAI");
assert.deepEqual(capitalise("são paulo"), "São Paulo");
assert.deepEqual(uniqueCapitalised(["acme", "Acme", " ", "united states"]), [
	"Acme",
	"United States",
]);
