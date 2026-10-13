import type { CompanyKind } from "@/types/company";
import type {
	ContactOutcome,
	RecruiterContactSummary,
} from "@/types/recruiterContact";

export interface AgencyRollup {
	roles: number;
	contacted: number;
	lastContactedOn: string | null;
	bestOutcome: ContactOutcome | null;
}

const OUTCOME_RANK: Record<ContactOutcome, number> = {
	no_reply: 1,
	rejected: 2,
	spoke: 3,
	submitted: 4,
};

export function isAgencyKind(kind: CompanyKind | undefined): boolean {
	return kind === "agency" || kind === "likely_agency";
}

export function bestOutcomeOf(
	a: ContactOutcome | null,
	b: ContactOutcome,
): ContactOutcome {
	return a && OUTCOME_RANK[a] >= OUTCOME_RANK[b] ? a : b;
}

export function agencyRollup(
	jobIds: readonly string[],
	summaries: Readonly<Record<string, RecruiterContactSummary>>,
): AgencyRollup {
	const rollup: AgencyRollup = {
		roles: jobIds.length,
		contacted: 0,
		lastContactedOn: null,
		bestOutcome: null,
	};
	for (const id of jobIds) {
		const summary = summaries[id];
		if (!summary) continue;
		rollup.contacted += 1;
		if (
			rollup.lastContactedOn === null ||
			summary.last_contacted_on > rollup.lastContactedOn
		) {
			rollup.lastContactedOn = summary.last_contacted_on;
		}
		rollup.bestOutcome = bestOutcomeOf(
			rollup.bestOutcome,
			summary.best_outcome,
		);
	}
	return rollup;
}
