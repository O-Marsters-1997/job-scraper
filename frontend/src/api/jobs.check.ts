import { setDemoData } from "../lib/demoData";
import { fetchJob, fetchJobs } from "./jobs";

setDemoData(true);
const first = await fetchJobs({ limit: 2 });
const second = await fetchJobs({ limit: 2, cursor: first.next_cursor });
if (
	first.items.length !== 2 ||
	!first.next_cursor ||
	second.items[0]?.ID === first.items[0]?.ID
) {
	throw new Error("job pages must be bounded and advance");
}
const detail = await fetchJob(first.items[0]?.ID ?? "");
if (!detail.Description)
	throw new Error("job detail must include full description");
setDemoData(false);
