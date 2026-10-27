import { useQueryClient } from "@tanstack/solid-query";
import { createMemo, createSignal, For, Show } from "solid-js";
import { FormFeedback } from "@/components/FormFeedback";
import { QuotaBar } from "@/components/QuotaBar";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import { useFormSubmit } from "@/hooks/useFormSubmit";
import { ApiError } from "../../../api/client";
import { keys } from "../../../api/keys";
import { useAiUsage } from "../../../hooks/useAiUsage";
import { useCVTemplates } from "../../../hooks/useCVTemplates";
import { useCreateDraft, useDiscardDraft } from "../../../hooks/useTailoring";
import { needsCv } from "../../../lib/applicationSelection";
import { loadedData } from "../../../lib/queryData";
import type { ApplicationWithDetails } from "../../../types/application";
import type { CV } from "../../../types/cv";

export function CreateDraftsDialog(props: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	apps: ApplicationWithDetails[];
}) {
	const queryClient = useQueryClient();
	const cvs = useCVTemplates();
	const usage = useAiUsage();
	const createDraft = useCreateDraft();
	const discardDraft = useDiscardDraft();

	const baseCvs = createMemo(() =>
		(loadedData(cvs)?.filter((c) => c.Visible) ?? []).sort(
			(a, b) => Number(b.Default) - Number(a.Default),
		),
	);
	const [picked, setPicked] = createSignal<CV | undefined>();
	const baseCv = () => picked() ?? baseCvs()[0];

	const [rowErrors, setRowErrors] = createSignal<Record<string, string>>({});
	const creatable = () => props.apps.filter(needsCv);

	const create = useFormSubmit(async () => {
		const cv = baseCv();
		if (!cv) return;
		const targets = creatable();
		const outcomes = await Promise.allSettled(
			targets.map(async (app) => {
				if (app.Draft) await discardDraft.mutateAsync(app.Draft.ID);
				await createDraft.mutateAsync({
					jobId: app.JobID,
					docId: cv.DocID,
					tabId: cv.TabID,
					mode: "generate",
				});
			}),
		);
		const errors: Record<string, string> = {};
		outcomes.forEach((outcome, i) => {
			const app = targets[i];
			if (!app || outcome.status === "fulfilled") return;
			const skipped =
				outcome.reason instanceof ApiError && outcome.reason.status === 409;
			errors[app.ID] = skipped
				? "Skipped, already has a CV"
				: "Couldn't create a CV for this job.";
		});
		setRowErrors(errors);
		await queryClient.invalidateQueries({ queryKey: keys.applications.all });
		if (Object.keys(errors).length === 0) props.onOpenChange(false);
	});

	return (
		<Dialog
			open={props.open}
			onOpenChange={(open) => {
				if (!open) setRowErrors({});
				props.onOpenChange(open);
			}}
		>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>Create CV drafts</DialogTitle>
					<p class="text-sm text-muted">
						Each Job gets a tailored CV generated in the background. Jobs that
						already have a CV are left alone.
					</p>
				</DialogHeader>
				<div class="flex flex-col gap-4">
					<Select
						options={baseCvs()}
						optionValue={(o) => `${o?.DocID}/${o?.TabID}`}
						optionTextValue={(o) => o?.Title ?? ""}
						disallowEmptySelection
						value={baseCv() ?? null}
						onChange={(cv) => cv && setPicked(cv)}
						itemComponent={(itemProps) => (
							<SelectItem item={itemProps.item}>
								<SelectItemLabel>
									{itemProps.item.rawValue?.Title}
								</SelectItemLabel>
							</SelectItem>
						)}
					>
						<SelectTrigger aria-label="Base CV">
							<Select.Value<CV>>
								{(state) => state.selectedOption()?.Title}
							</Select.Value>
						</SelectTrigger>
						<SelectContent />
					</Select>
					<ul class="flex max-h-64 flex-col divide-y divide-border overflow-y-auto rounded-lg border border-border">
						<For each={props.apps}>
							{(app) => (
								<li class="flex items-center justify-between gap-3 px-3 py-2 text-sm">
									<span
										class={
											needsCv(app)
												? "truncate text-foreground"
												: "truncate text-faint line-through"
										}
									>
										{app.JobTitle}
									</span>
									<Show when={!needsCv(app)}>
										<span class="shrink-0 text-xs text-muted">has CV</span>
									</Show>
									<Show when={rowErrors()[app.ID]}>
										{(message) => (
											<span class="shrink-0 text-xs text-destructive-strong">
												{message()}
											</span>
										)}
									</Show>
								</li>
							)}
						</For>
					</ul>
					<Show when={usage.data?.status === "ok" ? usage.data : undefined}>
						{(quota) => <QuotaBar quota={quota()} />}
					</Show>
					<FormFeedback error={create.error()} />
				</div>
				<DialogFooter class="border-t border-border pt-4">
					<Button
						type="button"
						variant="outline"
						onClick={() => props.onOpenChange(false)}
					>
						Cancel
					</Button>
					<Button
						type="button"
						loading={create.pending()}
						disabled={!baseCv() || creatable().length === 0}
						onClick={() => void create.submit()}
					>
						Create CV drafts
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
