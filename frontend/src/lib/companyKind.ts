import type { CompanyKind } from "@/types/company";
import type { Job } from "@/types/job";

export type PosterBucket = "direct" | "agency";

export function posterBucket(_kind: CompanyKind | undefined): PosterBucket {
	throw new Error("not implemented");
}

export function showsMaybeRecruiter(_job: Job): boolean {
	throw new Error("not implemented");
}
