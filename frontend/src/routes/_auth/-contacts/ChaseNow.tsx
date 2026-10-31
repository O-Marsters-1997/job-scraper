import { For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { formatChaseDate } from "@/lib/chase";
import type { ChaseRow } from "@/lib/chaseNow";
import type { ApplicationWithDetails } from "@/types/application";
import type { Contact, ContactSent } from "@/types/contact";
import { useClearChase } from "../../../hooks/useApplications";
import { ContactCard } from "./ContactCard";

export function ChaseNow(props: {
	rows: ChaseRow[];
	onSent: (c: Contact) => void;
	onReply: (c: Contact) => void;
	onDelete: (c: Contact) => void;
	onDraftSent: (sent: ContactSent) => void;
	onChased: (app: ApplicationWithDetails) => void;
}) {
	const clearChase = useClearChase();
	return (
		<section>
			<h2 class="mb-2 text-xs font-semibold uppercase tracking-wide text-faint">
				Chase now ({props.rows.length})
			</h2>
			<Show
				when={props.rows.length > 0}
				fallback={<p class="text-sm text-faint">Nothing to chase today</p>}
			>
				<div class="grid gap-3 md:grid-cols-2">
					<For each={props.rows}>
						{(row) =>
							row.kind === "contact" ? (
								<ContactCard
									contact={row.contact}
									applicationStatus={row.applicationStatus}
									onSent={() => props.onSent(row.contact)}
									onReply={() => props.onReply(row.contact)}
									onDelete={() => props.onDelete(row.contact)}
									onDraftSent={props.onDraftSent}
								/>
							) : (
								<Card class="flex flex-col gap-3 p-4">
									<div class="flex items-start justify-between gap-3">
										<div class="min-w-0">
											<h3 class="truncate text-sm font-semibold text-foreground">
												{row.app.JobTitle}
											</h3>
											<p class="truncate text-xs text-muted">
												{row.app.StatusName}
											</p>
										</div>
										<Show when={row.app.ChaseBy}>
											{(day) => (
												<span class="shrink-0 text-xs text-faint">
													Chase {formatChaseDate(day())}
												</span>
											)}
										</Show>
									</div>
									<div class="flex flex-wrap gap-2">
										<Button
											size="sm"
											loading={
												clearChase.isPending &&
												clearChase.variables === row.app.ID
											}
											onClick={() =>
												clearChase.mutate(row.app.ID, {
													onSuccess: () => props.onChased(row.app),
												})
											}
										>
											Mark chased
										</Button>
										<Show when={row.via}>
											{(via) => (
												<Button
													size="sm"
													variant="outline"
													onClick={() => props.onSent(via())}
												>
													Chase through {via().name}
												</Button>
											)}
										</Show>
									</div>
								</Card>
							)
						}
					</For>
				</div>
			</Show>
		</section>
	);
}
