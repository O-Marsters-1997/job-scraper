import {
	Content as SheetContentPrimitive,
	Portal as SheetPortal,
	Title as SheetTitle,
} from "@kobalte/core/dialog";
import { useNavigate } from "@tanstack/solid-router";
import { For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { Button } from "@/components/ui/button";
import { Dialog, DialogOverlay } from "@/components/ui/dialog";
import { useCVTemplates } from "@/hooks/useCVTemplates";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { useGoogleStatus } from "@/hooks/useGoogle";
import { useCreateDraft, useJobDrafts } from "@/hooks/useTailoring";
import { formatDate } from "@/lib/datetime";
import { sharePdf } from "@/lib/sharePdf";
import type { Job } from "@/types/job";
import { fetchCVPdf } from "../../api/cvTemplates";
import { fetchDraftPdf, fetchSuggestions } from "../../api/tailoring";

interface CvSheetProps {
	job: Pick<Job, "ID" | "CompanySlug">;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

export function CvSheet(props: CvSheetProps) {
	const google = useGoogleStatus();
	const cvs = useCVTemplates();
	const drafts = useJobDrafts(() => props.job.ID);

	let fetchPdf: () => Promise<ArrayBuffer> = async () => new ArrayBuffer(0);
	const share = useFormSubmit(async () =>
		sharePdf(await fetchPdf(), `cv-${props.job.CompanySlug}.pdf`),
	);
	const shareRow = (fetch: () => Promise<ArrayBuffer>) => () => {
		fetchPdf = fetch;
		void share.submit();
	};

	const navigate = useNavigate();
	const createDraft = useCreateDraft();
	let baseCv = { DocID: "", TabID: "" };
	const tailor = useFormSubmit(async () => {
		const { DocID: docId, TabID: tabId } = baseCv;
		const suggestions = await fetchSuggestions(props.job.ID, docId, tabId);
		const draft = await createDraft.mutateAsync({
			jobId: props.job.ID,
			docId,
			tabId,
			achievementIds: suggestions
				.filter((s) => s.preselected)
				.map((s) => s.achievementId),
		});
		await navigate({ to: "/tailoring/drafts/$id", params: { id: draft.id } });
	});

	const baseCvs = () => cvs.data?.filter((c) => c.Visible) ?? [];
	const keptDrafts = () =>
		drafts.data?.filter((d) => d.outcome === "kept") ?? [];

	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<SheetPortal>
				<DialogOverlay />
				<SheetContentPrimitive class="fixed inset-x-0 bottom-0 z-50 flex max-h-[80vh] flex-col gap-3 overflow-y-auto rounded-t-2xl border-t border-border bg-surface px-4 pt-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-2xl data-[expanded]:animate-in data-[closed]:animate-out data-[expanded]:slide-in-from-bottom-full data-[closed]:slide-out-to-bottom-full md:hidden">
					<SheetTitle class="text-base font-semibold text-foreground">
						Get a CV
					</SheetTitle>
					<FormFeedback error={share.error() ?? tailor.error()} />
					<Show
						when={google.data?.connected}
						fallback={
							<p class="text-sm text-muted">CVs need Google connected.</p>
						}
					>
						<Show
							when={baseCvs().length > 0 || keptDrafts().length > 0}
							fallback={<p class="text-sm text-muted">No CVs yet.</p>}
						>
							<For each={baseCvs()}>
								{(c) => (
									<fieldset aria-label={c.Title} class="flex min-w-0 gap-2">
										<Button
											variant="outline"
											class="flex-1 justify-between"
											disabled={share.pending()}
											onClick={shareRow(() => fetchCVPdf(c.DocID, c.TabID))}
										>
											<span>{c.Title}</span>
											<span class="text-xs text-faint">{c.SourceDoc}</span>
										</Button>
										<Button
											disabled={tailor.pending() || share.pending()}
											onClick={() => {
												baseCv = c;
												void tailor.submit();
											}}
										>
											{tailor.pending() ? "Tailoring…" : "Tailor"}
										</Button>
									</fieldset>
								)}
							</For>
							<For each={keptDrafts()}>
								{(d) => (
									<Button
										variant="outline"
										class="justify-between"
										disabled={share.pending()}
										onClick={shareRow(() => fetchDraftPdf(d.id))}
									>
										<span>Kept Draft</span>
										<span class="text-xs text-faint">
											{formatDate(d.createdAt)}
										</span>
									</Button>
								)}
							</For>
						</Show>
					</Show>
				</SheetContentPrimitive>
			</SheetPortal>
		</Dialog>
	);
}
