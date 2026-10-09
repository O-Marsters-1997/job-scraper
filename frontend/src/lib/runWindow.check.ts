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
assert(describeNextRun(null) === "Manual", "null is manual");
const now = new Date(2026, 9, 9, 8, 0);
assert(
	describeNextRun(new Date(2026, 9, 9, 9, 15).toISOString(), now) ===
		"Next run 09:15",
	"same day shows time",
);
assert(
	describeNextRun(new Date(2026, 9, 12, 9, 15).toISOString(), now) ===
		"Next run Mon 09:15",
	"later day shows weekday",
);
