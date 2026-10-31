import assert from "node:assert/strict";
import { isAgencyKind } from "./agencyRollup";

assert.equal(isAgencyKind("agency"), true);
assert.equal(isAgencyKind("likely_agency"), true);
assert.equal(isAgencyKind("direct"), false);
assert.equal(isAgencyKind(undefined), false);
