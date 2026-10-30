import {
	type Contact,
	type ContactCompany,
	type ContactInput,
	type ContactSent,
	contactCompanySchema,
	contactSchema,
	contactSentSchema,
	type MarkSentInput,
	type RecordReplyInput,
} from "../types/contact";
import { apiFetch, apiFetchVoid, jsonInit } from "./client";
import { mocked } from "./config";

export async function fetchContacts(): Promise<Contact[]> {
	return mocked(
		(db) => db.getContacts(),
		() => apiFetch("/contacts", contactSchema.array()),
	);
}

export async function fetchContactCompanies(): Promise<ContactCompany[]> {
	return mocked(
		(db) => db.getContactCompanies(),
		() => apiFetch("/contacts/companies", contactCompanySchema.array()),
	);
}

export async function createContact(input: ContactInput): Promise<Contact> {
	return mocked(
		(db) => db.createContact(input),
		() => apiFetch("/contacts", contactSchema, jsonInit("POST", input)),
	);
}

export async function markContactSent(
	id: string,
	input: MarkSentInput,
): Promise<ContactSent> {
	return mocked(
		(db) => db.markContactSent(id, input),
		() =>
			apiFetch(
				`/contacts/${encodeURIComponent(id)}/sent`,
				contactSentSchema,
				jsonInit("POST", input),
			),
	);
}

export async function recordContactReply(
	id: string,
	input: RecordReplyInput,
): Promise<Contact> {
	return mocked(
		(db) => db.recordContactReply(id, input),
		() =>
			apiFetch(
				`/contacts/${encodeURIComponent(id)}/reply`,
				contactSchema,
				jsonInit("POST", input),
			),
	);
}

export async function deleteContact(id: string): Promise<void> {
	return mocked(
		(db) => db.deleteContact(id),
		() =>
			apiFetchVoid(`/contacts/${encodeURIComponent(id)}`, { method: "DELETE" }),
	);
}
