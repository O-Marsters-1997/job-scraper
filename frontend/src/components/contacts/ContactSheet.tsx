import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { DownloadPdfButton } from "@/components/tailoring/DownloadPdfButton";
import { Button } from "@/components/ui/button";
import {
	Sheet,
	SheetClose,
	SheetContent,
	SheetHeader,
	SheetTitle,
} from "@/components/ui/sheet";
import { useJobDrafts } from "@/hooks/useTailoring";
import { formatChaseDate } from "@/lib/chase";
import { openReachOut } from "@/lib/contactStatus";
import { eventLabel } from "@/lib/eventLabel";
import { loadedData } from "@/lib/queryData";
import { Drafter } from "@/routes/_auth/-contacts/Drafter";
import type { Contact, ContactSent } from "@/types/contact";

function KeptCvDownload(props: { jobId: string }) {
	const drafts = useJobDrafts(() => props.jobId);
	const kept = () => loadedData(drafts)?.find((d) => d.outcome === "kept");
	return (
		<Show when={kept()}>
			{(draft) => (
				<DownloadPdfButton
					draftId={draft().id}
					jobId={props.jobId}
					variant="outline"
					label="Download CV"
				/>
			)}
		</Show>
	);
}

export function ContactSheet(props: {
	contact: Contact | undefined;
	onClose: () => void;
	onDraftSent: (sent: ContactSent) => void;
	onDelete: (c: Contact) => void;
}) {
	return (
		<Sheet open={!!props.contact} onOpenChange={(o) => !o && props.onClose()}>
			<SheetContent class="w-full max-w-md overflow-y-auto">
				<Show when={props.contact}>
					{(c) => (
						<>
							<SheetHeader>
								<div class="min-w-0">
									<SheetTitle class="truncate">{c().name}</SheetTitle>
									<p class="truncate text-xs text-muted">
										{[c().role, c().company_name].filter(Boolean).join(" · ")}
									</p>
								</div>
								<SheetClose
									as={Button}
									variant="ghost"
									size="sm"
									aria-label="Close"
								>
									<Icon name="x" />
								</SheetClose>
							</SheetHeader>
							<div class="flex flex-col gap-4 px-6 py-4">
								<div class="flex flex-wrap gap-x-4 gap-y-1 text-sm">
									<Show when={c().email}>
										<a class="text-primary" href={`mailto:${c().email}`}>
											{c().email}
										</a>
									</Show>
									<Show when={c().linkedin_url.startsWith("http")}>
										<a
											class="text-primary"
											href={c().linkedin_url}
											target="_blank"
											rel="noreferrer"
										>
											LinkedIn
										</a>
									</Show>
								</div>
								<Show when={openReachOut(c())}>
									{(r) => (
										<div class="flex flex-wrap items-center justify-between gap-2">
											<p class="text-sm text-foreground">
												Reaching out about{" "}
												<span class="font-medium">{r().job_title}</span>
											</p>
											<KeptCvDownload jobId={r().job_id} />
										</div>
									)}
								</Show>
								<Show when={c().reply}>
									<blockquote class="border-l-2 border-border pl-3 text-sm text-muted">
										{c().reply}
									</blockquote>
								</Show>
								<Show when={c().stage !== "closed"}>
									<Drafter contact={c()} onSent={props.onDraftSent} />
								</Show>
								<ol class="flex flex-col gap-1 border-t border-border pt-3 text-xs text-muted">
									<For each={c().history}>
										{(e) => (
											<li>
												<span class="text-faint">
													{formatChaseDate(e.occurred_on)}
												</span>{" "}
												{eventLabel(e)}
											</li>
										)}
									</For>
								</ol>
								<div>
									<Button
										size="sm"
										variant="ghost"
										onClick={() => props.onDelete(c())}
									>
										Delete
									</Button>
								</div>
							</div>
						</>
					)}
				</Show>
			</SheetContent>
		</Sheet>
	);
}
