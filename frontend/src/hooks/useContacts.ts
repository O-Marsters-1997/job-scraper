import { createQuery, queryOptions } from "@tanstack/solid-query";
import {
	createContact,
	deleteContact,
	fetchContactCompanies,
	fetchContacts,
	markContactSent,
	recordContactReply,
} from "../api/contacts";
import { keys } from "../api/keys";
import type { MarkSentInput, RecordReplyInput } from "../types/contact";
import { useInvalidatingMutation } from "./useInvalidatingMutation";

export const contactsQueryOptions = queryOptions({
	queryKey: keys.contacts,
	queryFn: fetchContacts,
});

export function useContacts() {
	return createQuery(() => contactsQueryOptions);
}

export const contactCompaniesQueryOptions = queryOptions({
	queryKey: keys.contactCompanies,
	queryFn: fetchContactCompanies,
});

export function useContactCompanies() {
	return createQuery(() => contactCompaniesQueryOptions);
}

export function useCreateContact() {
	return useInvalidatingMutation(createContact, [keys.contacts]);
}

export function useMarkContactSent() {
	return useInvalidatingMutation(
		(v: { id: string } & MarkSentInput) => {
			const { id, ...input } = v;
			return markContactSent(id, input);
		},
		[keys.contacts, keys.applications.all],
	);
}

export function useRecordContactReply() {
	return useInvalidatingMutation(
		(v: { id: string } & RecordReplyInput) => {
			const { id, ...input } = v;
			return recordContactReply(id, input);
		},
		[keys.contacts],
	);
}

export function useDeleteContact() {
	return useInvalidatingMutation(deleteContact, [keys.contacts]);
}
