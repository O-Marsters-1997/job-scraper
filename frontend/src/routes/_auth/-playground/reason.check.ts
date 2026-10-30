import assert from "node:assert/strict";
import { buildReason } from "./reason";

assert.equal(buildReason("", []), "");
assert.equal(buildReason("  fine ", [{ label: "A", note: " " }]), "fine");
assert.equal(
	buildReason("Too high", [
		{ label: "Remote UK", note: "says hybrid 3 days" },
		{ label: "B", note: "" },
	]),
	"Too high\nRemote UK → says hybrid 3 days",
);
assert.equal(buildReason("", [{ label: "A", note: "x" }]), "A → x");
