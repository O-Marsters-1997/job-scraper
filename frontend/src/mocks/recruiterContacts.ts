import { bestOutcomeOf } from "@/lib/agencyRollup";
import type {
	LogRecruiterContactPayload,
	RecruiterContact,
	RecruiterContactSummary,
} from "@/types/recruiterContact";

let contacts: RecruiterContact[] = [];

export function getRecruiterContacts(jobId: string): RecruiterContact[] {
	return contacts
		.filter((c) => c.job_id === jobId)
		.sort(
			(a, b) =>
				b.contacted_on.localeCompare(a.contacted_on) ||
				b.created_at.localeCompare(a.created_at),
		);
}

export function logRecruiterContact(
	jobId: string,
	payload: LogRecruiterContactPayload,
): RecruiterContact {
	const contact: RecruiterContact = {
		id: `contact-${contacts.length + 1}`,
		job_id: jobId,
		...payload,
		created_at: new Date().toISOString(),
	};
	contacts = [...contacts, contact];
	return contact;
}

export function getRecruiterContactSummaries(
	jobIds: string[],
): Record<string, RecruiterContactSummary> {
	const out: Record<string, RecruiterContactSummary> = {};
	for (const c of contacts) {
		if (!jobIds.includes(c.job_id)) continue;
		const prev = out[c.job_id];
		out[c.job_id] = {
			count: (prev?.count ?? 0) + 1,
			last_contacted_on:
				prev && prev.last_contacted_on > c.contacted_on
					? prev.last_contacted_on
					: c.contacted_on,
			best_outcome: bestOutcomeOf(prev?.best_outcome ?? null, c.outcome),
		};
	}
	return out;
}
