import { createEffect } from "solid-js";
import { recordImpressions } from "@/api/events";
import type { ImpressionBatchInput } from "../types/event";
import type { Job } from "../types/job";

const MAX_BATCH = 50;

export function useJobImpressions(
	items: () => Job[],
	firstPosition: () => number,
	sort: () => ImpressionBatchInput["sort"],
) {
	let sent = "";
	createEffect(() => {
		const jobs = items();
		const key = `${sort()}:${firstPosition()}:${jobs.map((j) => j.ID).join(",")}`;
		if (jobs.length === 0 || key === sent) return;
		sent = key;
		recordImpressions({
			surface: "list",
			sort: sort(),
			items: jobs.slice(0, MAX_BATCH).map((j, i) => ({
				job_id: j.ID,
				position: firstPosition() + i,
			})),
		});
	});
}
