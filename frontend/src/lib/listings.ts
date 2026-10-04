import type { Job, JobListing } from "@/types/job";
import type { SourceInfo } from "@/types/source";
import { titleCase } from "./utils";

export function otherListings(job: Job): JobListing[] {
	return (job.Listings ?? []).filter((listing) => listing.url !== job.URL);
}

export function sourceLabel(
	name: string,
	sources: readonly SourceInfo[] | undefined,
): string {
	return sources?.find((s) => s.name === name)?.label ?? titleCase(name);
}
