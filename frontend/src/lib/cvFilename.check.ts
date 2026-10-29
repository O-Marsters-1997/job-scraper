import assert from "node:assert/strict";
import { cvFilename } from "./cvFilename";

const values = {
	name: "olly",
	company: "acme-corp",
	role: "Senior Engineer",
	date: "2026-10-29",
};

assert.equal(cvFilename("cv-{company}", values), "cv-acme-corp.pdf");
assert.equal(
	cvFilename("{name}_{role}_{date}", values),
	"olly_Senior-Engineer_2026-10-29.pdf",
);
assert.equal(cvFilename("", values), "cv-acme-corp.pdf");
assert.equal(cvFilename("   ", values), "cv-acme-corp.pdf");
assert.equal(cvFilename("{team}-{company}", values), "{team}-acme-corp.pdf");
assert.equal(
	cvFilename("{role}", { ...values, role: 'QA/Test: "Lead"?' }),
	"QA-Test-Lead.pdf",
);
assert.equal(cvFilename("a/b\\c*d", values), "a-b-c-d.pdf");
assert.equal(cvFilename("{company}", { ...values, company: "" }), "cv.pdf");
