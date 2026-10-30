import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { AddContactDialog } from "@/components/contacts/AddContactDialog";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { contactGroups, groupContacts } from "@/lib/contactGroups";
import type { Contact } from "@/types/contact";
import {
	contactsQueryOptions,
	useContacts,
	useDeleteContact,
} from "../../hooks/useContacts";
import { queryClient } from "../../lib/queryClient";
import { ContactCard } from "./-contacts/ContactCard";
import { MarkSentDialog, ReplyDialog } from "./-contacts/ContactDialogs";

export const Route = createFileRoute("/_auth/contacts")({
	loader: () => queryClient.prefetchQuery(contactsQueryOptions),
	component: ContactsPage,
});

function ContactsPage() {
	const contacts = useContacts();
	const remove = useDeleteContact();
	const [adding, setAdding] = createSignal(false);
	const [sending, setSending] = createSignal<Contact | undefined>();
	const [replying, setReplying] = createSignal<Contact | undefined>();

	return (
		<div class="px-7 py-6">
			<PageHeading title="Contacts" subtitle="People you're reaching out to">
				<Button onClick={() => setAdding(true)}>Add contact</Button>
			</PageHeading>
			<AddContactDialog open={adding()} onOpenChange={setAdding} />
			<MarkSentDialog
				contact={sending()}
				onClose={() => setSending(undefined)}
			/>
			<ReplyDialog
				contact={replying()}
				onClose={() => setReplying(undefined)}
			/>

			<QueryBoundary query={contacts}>
				{(all) => (
					<Show
						when={all().length > 0}
						fallback={
							<p class="text-sm text-faint">
								No contacts yet. Add a recruiter or hiring manager to start.
							</p>
						}
					>
						<div class="flex flex-col gap-6">
							<For each={contactGroups}>
								{(group) => (
									<Show when={groupContacts(all())[group.key]}>
										{(items) => (
											<Show when={items().length > 0}>
												<section>
													<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-faint">
														{group.label} ({items().length})
													</h2>
													<div class="grid gap-3 md:grid-cols-2">
														<For each={items()}>
															{(c) => (
																<ContactCard
																	contact={c}
																	onSent={() => setSending(c)}
																	onReply={() => setReplying(c)}
																	onDelete={() => {
																		if (
																			confirm(
																				`Delete ${c.name} and their history?`,
																			)
																		) {
																			remove.mutate(c.id);
																		}
																	}}
																/>
															)}
														</For>
													</div>
												</section>
											</Show>
										)}
									</Show>
								)}
							</For>
						</div>
					</Show>
				)}
			</QueryBoundary>
		</div>
	);
}
