import assert from "node:assert/strict";
import type { LayoutBlock, LayoutRun } from "@/types/tailoring";
import { cloneRoleHeading } from "./roleHeading";

const run = (text: string, bold = false): LayoutRun => ({
	text,
	font: "Calibri",
	size: 11,
	bold,
	italic: false,
	underline: false,
	color: "",
	link: "",
});

const heading = (...runs: LayoutRun[]) => ({ runs }) as LayoutBlock;
const from = { title: "Engineer", employer: "Acme" };
const to = {
	title: "Analyst",
	employer: "Widgets",
	startDate: "2016-04-01",
	endDate: "2018-05-31",
};
const textOf = (b: LayoutBlock) => b.runs.map((r) => r.text);

{
	const got = cloneRoleHeading(
		heading(run("Engineer, Acme\t", true), run("2020 - 2022")),
		from,
		to,
	);
	assert.deepEqual(textOf(got), ["Analyst, Widgets\t", "2016 - 2018"]);
	assert.equal(got.runs[0]?.bold, true);
}

{
	const got = cloneRoleHeading(
		heading(run("Engineer, Acme\t"), run("Apr 2020 – May 2022")),
		from,
		to,
	);
	assert.deepEqual(textOf(got), ["Analyst, Widgets\t", "Apr 2016 – May 2018"]);
}

{
	const got = cloneRoleHeading(
		heading(run("Engineer, Acme\t"), run("March 2020 - Present")),
		from,
		{ ...to, endDate: null },
	);
	assert.deepEqual(textOf(got), ["Analyst, Widgets\t", "April 2016 - Present"]);
}

{
	const got = cloneRoleHeading(
		heading(run("Backend lead", true), run("\t2020 - 2022")),
		from,
		to,
	);
	assert.deepEqual(textOf(got), ["Analyst, Widgets", "\t2016 - 2018"]);
}
