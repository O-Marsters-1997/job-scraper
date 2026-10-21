import type { Job } from "@/types/job";

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
