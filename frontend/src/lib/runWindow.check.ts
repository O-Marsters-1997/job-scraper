import {
	DEFAULT_RUN_WINDOW,
	describeNextRun,
	draftError,
	draftFrom,
	toRunWindow,
} from "./runWindow";

const assert = (cond: boolean, msg: string) => {
	if (!cond) throw new Error(msg);
};

const draft = draftFrom(DEFAULT_RUN_WINDOW, true);
assert(
	JSON.stringify(toRunWindow(draft)) === JSON.stringify(DEFAULT_RUN_WINDOW),
	"default draft round-trips",
);
assert(
	toRunWindow({ ...draft, automatic: false }) === null,
	"manual sends null",
);
assert(draftError({ ...draft, weekdays: [] }) !== null, "empty days invalid");
assert(draftError({ ...draft, end: "07:00" }) !== null, "inverted invalid");
assert(
	draftError({ ...draft, automatic: false, weekdays: [] }) === null,
	"manual skips validation",
);
assert(describeNextRun(null, "Europe/London") === "Manual", "null is manual");
const now = new Date("2026-10-09T07:00:00Z");
assert(
	describeNextRun("2026-10-09T08:15:00Z", "Europe/London", now) ===
		"Next run 09:15",
	"same day shows time",
);
assert(
	describeNextRun("2026-10-12T08:15:00Z", "Europe/London", now) ===
		"Next run Mon 12 Oct 09:15",
	"later day shows weekday",
);
