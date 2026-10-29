import assert from "node:assert/strict";
import { matchesApplication } from "./applicationSearch";

const app = {
	JobTitle: "Senior Go Engineer",
	JobCompanySlug: "acme-ltd",
	JobLocation: "Leeds, UK",
};

assert.equal(matchesApplication(app, ""), true);
assert.equal(matchesApplication(app, "   "), true);
assert.equal(matchesApplication(app, "go eng"), true);
assert.equal(matchesApplication(app, "ACME"), true);
assert.equal(matchesApplication(app, " leeds "), true);
assert.equal(matchesApplication(app, "london"), false);
