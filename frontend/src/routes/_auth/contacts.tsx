import { createFileRoute } from "@tanstack/solid-router";
import { createEffect, createSignal, on, onCleanup, Show } from "solid-js";
import { AddContactDialog } from "@/components/contacts/AddContactDialog";
import { ContactSection } from "@/components/contacts/ContactSection";
import { ContactSheet } from "@/components/contacts/ContactSheet";
import { TemplatesPanel } from "@/components/contacts/TemplatesPanel";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { contactSections } from "@/lib/contactStatus";
import { loadedData } from "@/lib/queryData";
import type { Contact, ContactSent } from "@/types/contact";
import {
	contactCompaniesQueryOptions,
	contactsQueryOptions,
	useContactCompanies,
	useContacts,
	useDeleteContact,
} from "../../hooks/useContacts";
import { displayName } from "../../lib/contactName";
import { queryClient } from "../../lib/queryClient";
import { CompaniesView } from "./-contacts/CompaniesView";
import { MarkSentDialog, ReplyDialog } from "./-contacts/ContactDialogs";
import { ReceiptToast } from "./-contacts/ReceiptToast";

const HIGHLIGHT_MS = 3000;

export const Route = createFileRoute("/_auth/contacts")({
	loader: () =>
		Promise.all([
			queryClient.prefetchQuery(contactsQueryOptions),
			queryClient.prefetchQuery(contactCompaniesQueryOptions),
		]),
	component: ContactsPage,
});

function ContactsPage() {
	const contacts = useContacts();
	const remove = useDeleteContact();
	const companies = useContactCompanies();
	const [tab, setTab] = createSignal("people");
	const [adding, setAdding] = createSignal(false);
	const [addingAt, setAddingAt] = createSignal<
		{ id: string | null; name: string } | undefined
	>();
	const [sending, setSending] = createSignal<Contact | undefined>();
	const [replying, setReplying] = createSignal<Contact | undefined>();
	const [openId, setOpenId] = createSignal<string | undefined>();
	const [highlightId, setHighlightId] = createSignal<string | undefined>();
	const [receipt, setReceipt] = createSignal<ContactSent | undefined>();
	const openContact = () =>
		loadedData(contacts)?.find((c) => c.id === openId());
	const deleteContact = (c: Contact) => {
		if (!confirm(`Delete ${displayName(c)} and their history?`)) return;
		setOpenId(undefined);
		remove.mutate(c.id);
	};
	const sent = (s: ContactSent) => {
		setOpenId(undefined);
		setReceipt(s);
		setHighlightId(s.contact.id);
	};
	createEffect(
		on(highlightId, (id) => {
			if (!id) return;
			const timer = setTimeout(() => setHighlightId(undefined), HIGHLIGHT_MS);
			onCleanup(() => clearTimeout(timer));
		}),
	);

	return (
		<div class="px-7 py-6">
			<PageHeading title="Contacts" subtitle="People you're reaching out to">
				<Button
					onClick={() => {
						setAddingAt(undefined);
						setAdding(true);
					}}
				>
					Add contact
				</Button>
			</PageHeading>
			<AddContactDialog
				open={adding()}
				onOpenChange={setAdding}
				company={addingAt()}
			/>
			<MarkSentDialog
				contact={sending()}
				onClose={() => setSending(undefined)}
			/>
			<ReplyDialog
				contact={replying()}
				onClose={() => setReplying(undefined)}
			/>
			<ContactSheet
				contact={openContact()}
				onClose={() => setOpenId(undefined)}
				onDraftSent={sent}
				onDelete={deleteContact}
			/>

			<ReceiptToast sent={receipt()} onDismiss={() => setReceipt(undefined)} />

			<Tabs value={tab()} onChange={setTab}>
				<TabsList variant="line" class="mb-3">
					<TabsTrigger value="people">People</TabsTrigger>
					<TabsTrigger value="companies">Companies</TabsTrigger>
					<TabsTrigger value="templates">Templates</TabsTrigger>
				</TabsList>
				<TabsContent value="templates">
					<TemplatesPanel />
				</TabsContent>
				<TabsContent value="companies">
					<QueryBoundary query={companies}>
						{(list) => (
							<CompaniesView
								companies={list()}
								contacts={loadedData(contacts) ?? []}
								onOpenPerson={setSending}
								onAdd={(c) => {
									setAddingAt(c);
									setAdding(true);
								}}
							/>
						)}
					</QueryBoundary>
				</TabsContent>
				<TabsContent value="people">
					<QueryBoundary query={contacts}>
						{(all) => {
							const sections = () => contactSections(all());
							const open = (c: Contact) => setOpenId(c.id);
							return (
								<Show
									when={all().length > 0}
									fallback={
										<p class="text-sm text-faint">
											No contacts yet. Add a recruiter or hiring manager to
											start.
										</p>
									}
								>
									<div class="flex flex-col gap-6">
										<ContactSection
											title="To contact"
											kind="toContact"
											contacts={sections().toContact}
											highlightId={highlightId()}
											onOpen={open}
											onReply={setReplying}
										/>
										<ContactSection
											title="Waiting for a reply"
											kind="waiting"
											contacts={sections().waiting}
											highlightId={highlightId()}
											onOpen={open}
											onReply={setReplying}
										/>
										<ContactSection
											title="Everyone else"
											kind="rest"
											contacts={sections().rest}
											highlightId={highlightId()}
											onOpen={open}
											onReply={setReplying}
										/>
									</div>
								</Show>
							);
						}}
					</QueryBoundary>
				</TabsContent>
			</Tabs>
		</div>
	);
}
