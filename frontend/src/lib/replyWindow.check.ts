import assert from "node:assert/strict";
import { addWorkingDays, parseReplyWindow } from "./replyWindow";

const ymd = (d: Date) =>
	`${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
const local = (s: string) => new Date(`${s}T12:00:00`);

// 2026-10-02 is a Friday
assert.equal(ymd(addWorkingDays(local("2026-10-02"), 1)), "2026-10-05");
assert.equal(ymd(addWorkingDays(local("2026-10-02"), 5)), "2026-10-09");
assert.equal(ymd(addWorkingDays(local("2026-10-03"), 1)), "2026-10-05");
assert.equal(ymd(addWorkingDays(local("2026-10-04"), 1)), "2026-10-05");
assert.equal(ymd(addWorkingDays(local("2026-10-03"), 5)), "2026-10-09");
assert.equal(ymd(addWorkingDays(local("2026-10-01"), 10)), "2026-10-15");
assert.equal(ymd(addWorkingDays(local("2026-10-02"), 23)), "2026-11-04");
assert.equal(ymd(addWorkingDays(local("2026-10-05"), 60)), "2026-12-28");
const sat = local("2026-10-03");
assert.equal(ymd(addWorkingDays(sat, 0)), "2026-10-05");
assert.equal(ymd(sat), "2026-10-03");

assert.deepEqual(parseReplyWindow(""), { days: null });
assert.deepEqual(parseReplyWindow("  "), { days: null });
assert.deepEqual(parseReplyWindow("1"), { days: 1 });
assert.deepEqual(parseReplyWindow("60"), { days: 60 });
for (const bad of ["0", "61", "-3", "2.5", "abc"]) {
	assert.ok(parseReplyWindow(bad).error, `${bad} should be rejected`);
}
