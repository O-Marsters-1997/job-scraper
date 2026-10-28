import type { Job, JobInsights } from "@/types/job";
import { dayKey } from "./datetime";

export function firstFoundBy(job: Job): string {
	if (job.DiscoveredVia) return job.DiscoveredVia;
	const earliest = (job.Listings ?? []).reduce<
		NonNullable<Job["Listings"]>[number] | undefined
	>(
		(best, l) => (!best || l.first_seen_at < best.first_seen_at ? l : best),
		undefined,
	);
	return earliest?.source ?? job.Source;
}

export function buildInsights(jobs: Job[]): JobInsights {
	const days = new Map<string, number>();
	const sources = new Map<string, { count: number; scores: number[] }>();
	const found = new Map<string, number>();
	const scoreBands = [0, 0, 0, 0, 0];
	let scored = 0;
	for (const job of jobs) {
		const day = dayKey(job.ScrapedAt);
		days.set(day, (days.get(day) ?? 0) + 1);
		const source = sources.get(job.Source) ?? { count: 0, scores: [] };
		source.count++;
		sources.set(job.Source, source);
		if (job.SuitabilityScore != null) {
			scored++;
			source.scores.push(job.SuitabilityScore);
			const band = Math.min(Math.floor(job.SuitabilityScore / 20), 4);
			scoreBands[band] = (scoreBands[band] ?? 0) + 1;
		}
		if (job.Band === "good" || job.Band === "great") {
			const who = firstFoundBy(job);
			found.set(who, (found.get(who) ?? 0) + 1);
		}
	}
	return {
		total: jobs.length,
		scored,
		daily: [...days]
			.map(([day, count]) => ({ day, count }))
			.sort((a, b) => a.day.localeCompare(b.day)),
		sources: [...sources]
			.map(([source, { count, scores }]) => ({
				source,
				count,
				scored: scores.length,
				avgScore: scores.length
					? Math.round(scores.reduce((a, b) => a + b, 0) / scores.length)
					: 0,
			}))
			.sort((a, b) => b.count - a.count),
		scoreBands,
		foundBy: [...found]
			.map(([name, count]) => ({ name, count }))
			.sort((a, b) => b.count - a.count),
	};
}
