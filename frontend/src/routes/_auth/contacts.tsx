import { createFileRoute } from "@tanstack/solid-router";
import { createSignal, For, Show } from "solid-js";
import { AddContactDialog } from "@/components/contacts/AddContactDialog";
import { TemplatesPanel } from "@/components/contacts/TemplatesPanel";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { chaseNowRows } from "@/lib/chaseNow";
import { contactGroups, groupContacts } from "@/lib/contactGroups";
import { loadedData } from "@/lib/queryData";
import type { ApplicationWithDetails } from "@/types/application";
import type { Contact } from "@/types/contact";
import {
	chasesQueryOptions,
	useApplications,
	useChases,
} from "../../hooks/useApplications";
import {
	contactCompaniesQueryOptions,
	contactsQueryOptions,
	useContactCompanies,
	useContacts,
	useDeleteContact,
} from "../../hooks/useContacts";
import { queryClient } from "../../lib/queryClient";
import { SetAnotherChase } from "./-applications/SetAnotherChase";
import { ChaseNow } from "./-contacts/ChaseNow";
import { CompaniesView } from "./-contacts/CompaniesView";
import { ContactCard } from "./-contacts/ContactCard";
import { MarkSentDialog, ReplyDialog } from "./-contacts/ContactDialogs";

export const Route = createFileRoute("/_auth/contacts")({
	loader: () =>
		Promise.all([
			queryClient.prefetchQuery(contactsQueryOptions),
			queryClient.prefetchQuery(chasesQueryOptions),
			queryClient.prefetchQuery(contactCompaniesQueryOptions),
		]),
	component: ContactsPage,
});

function ContactsPage() {
	const contacts = useContacts();
	const chases = useChases();
	const apps = useApplications();
	const remove = useDeleteContact();
	const companies = useContactCompanies();
	const [tab, setTab] = createSignal("people");
	const [adding, setAdding] = createSignal(false);
	const [addingAt, setAddingAt] = createSignal<
		{ id: string | null; name: string } | undefined
	>();
	const [sending, setSending] = createSignal<Contact | undefined>();
	const [replying, setReplying] = createSignal<Contact | undefined>();
	const [chasedApps, setChasedApps] = createSignal<ApplicationWithDetails[]>(
		[],
	);
	const deleteContact = (c: Contact) => {
		if (confirm(`Delete ${c.name} and their history?`)) remove.mutate(c.id);
	};

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

			<For each={chasedApps()}>
				{(app) => (
					<SetAnotherChase
						applicationId={app.ID}
						title={app.JobTitle}
						onDone={() =>
							setChasedApps((prev) => prev.filter((p) => p.ID !== app.ID))
						}
					/>
				)}
			</For>

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
							const rows = () =>
								chaseNowRows(
									all(),
									loadedData(chases) ?? [],
									new Date(),
									loadedData(apps) ?? [],
								);
							const chaseIds = () =>
								new Set(
									rows().flatMap((r) =>
										r.kind === "contact" ? [r.contact.id] : [],
									),
								);
							const grouped = () =>
								groupContacts(all().filter((c) => !chaseIds().has(c.id)));
							return (
								<div class="flex flex-col gap-6">
									<ChaseNow
										rows={rows()}
										onSent={setSending}
										onReply={setReplying}
										onDelete={deleteContact}
										onChased={(app) => setChasedApps((prev) => [...prev, app])}
									/>
									<Show
										when={all().length > 0}
										fallback={
											<p class="text-sm text-faint">
												No contacts yet. Add a recruiter or hiring manager to
												start.
											</p>
										}
									>
										<For each={contactGroups}>
											{(group) => (
												<Show when={grouped()[group.key].length > 0}>
													<section>
														<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-faint">
															{group.label} ({grouped()[group.key].length})
														</h2>
														<div class="grid gap-3 md:grid-cols-2">
															<For each={grouped()[group.key]}>
																{(c) => (
																	<ContactCard
																		contact={c}
																		onSent={() => setSending(c)}
																		onReply={() => setReplying(c)}
																		onDelete={() => deleteContact(c)}
																	/>
																)}
															</For>
														</div>
													</section>
												</Show>
											)}
										</For>
									</Show>
								</div>
							);
						}}
					</QueryBoundary>
				</TabsContent>
			</Tabs>
		</div>
	);
}
