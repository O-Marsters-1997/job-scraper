import type { Job } from "@/types/job";
import type {
	ContactOutcome,
	RecruiterContactSummary,
} from "@/types/recruiterContact";

export interface AgencyRollup {
	roleCount: number;
	contactedRoleCount: number;
	lastContactedOn: string | undefined;
	bestOutcome: ContactOutcome | undefined;
}

export function agencyRollup(
	_jobs: Job[],
	_summaries: Record<string, RecruiterContactSummary>,
): AgencyRollup {
	throw new Error("not implemented");
}
