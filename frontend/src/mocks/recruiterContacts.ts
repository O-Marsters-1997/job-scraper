import type {
	LogRecruiterContactPayload,
	RecruiterContact,
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
