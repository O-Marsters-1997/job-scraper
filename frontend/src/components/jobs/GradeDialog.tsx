import {
	createEffect,
	createSignal,
	For,
	onMount,
	Show,
	untrack,
} from "solid-js";
import { fetchGrade } from "@/api/grades";
import { keys } from "@/api/keys";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { useGrade, useSetGrade } from "@/hooks/useGrades";
import { BAND_LABEL, gradeDirection } from "@/lib/band";
import { planSaves } from "@/lib/gradeBatch";
import { queryClient } from "@/lib/queryClient";
import { cn, titleCase } from "@/lib/utils";
import {
	GRADE_LABEL,
	GRADES,
	type Grade,
	type GradeReason,
	type GradeValue,
} from "@/types/grade";
import type { Job } from "@/types/job";
import { BandChip } from "./BandChip";
import { GradeChips } from "./GradeChips";

interface Props {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	jobs: Job[];
	onSaved: (savedIds: string[], priors: Record<string, Grade | null>) => void;
}

export function GradeDialog(props: Props) {
	return (
		<Dialog open={props.open} onOpenChange={props.onOpenChange}>
			<Show when={props.open && props.jobs.length > 0}>
				<GradeForm
					jobs={untrack(() => props.jobs)}
					onDone={(saved, priors) => {
						props.onSaved(saved, priors);
						props.onOpenChange(false);
					}}
					onCancel={() => props.onOpenChange(false)}
				/>
			</Show>
		</Dialog>
	);
}

function GradeForm(props: {
	jobs: Job[];
	onDone: Props["onSaved"];
	onCancel: () => void;
}) {
	const jobs = untrack(() => props.jobs);
	const bulk = jobs.length > 1;
	const set = useSetGrade();
	const [batch, setBatch] = createSignal(jobs);
	const [index, setIndex] = createSignal(0);
	const [batchGrade, setBatchGrade] = createSignal<GradeValue | undefined>();
	const [overrides, setOverrides] = createSignal<Record<string, GradeValue>>(
		{},
	);
	const [reasons, setReasons] = createSignal<Record<string, GradeReason[]>>({});
	const [priors, setPriors] = createSignal<Record<string, Grade | null>>();
	const [saved, setSaved] = createSignal<string[]>([]);
	const [pending, setPending] = createSignal(false);
	const [failed, setFailed] = createSignal(false);

	const job = () => batch()[index()] as Job;
	const effective = (id: string) => overrides()[id] ?? batchGrade();
	const jobReasons = () => reasons()[job().ID] ?? [];
	const clearOverride = () =>
		setOverrides(({ [job().ID]: _, ...rest }) => rest);

	const existing = useGrade(() => (bulk ? "" : (jobs[0]?.ID ?? "")));
	createEffect(() => {
		const g = existing.data;
		if (g && untrack(batchGrade) === undefined) {
			setBatchGrade(g.grade);
			setReasons({ [g.jobId]: g.reasons });
		}
	});

	onMount(async () => {
		if (!bulk) return;
		const entries = await Promise.all(
			jobs.map(async (j) => {
				try {
					const prior = await queryClient.fetchQuery({
						queryKey: keys.grade(j.ID),
						queryFn: () => fetchGrade(j.ID),
					});
					return [[j.ID, prior] as const];
				} catch {
					return [];
				}
			}),
		);
		setPriors(Object.fromEntries(entries.flat()));
	});

	const setJobGrade = (value: GradeValue) => {
		if (!bulk) return setBatchGrade(value);
		if (value === batchGrade()) return clearOverride();
		setOverrides((o) => ({ ...o, [job().ID]: value }));
	};
	const go = (by: number) =>
		setIndex((i) => Math.min(batch().length - 1, Math.max(0, i + by)));

	const onKeyDown = (e: KeyboardEvent) => {
		const target = e.target as HTMLElement;
		if (
			e.metaKey ||
			e.ctrlKey ||
			e.altKey ||
			target.closest("[data-no-hotkeys]")
		)
			return;
		const grade = GRADES[Number(e.key) - 1];
		if (grade) {
			e.preventDefault();
			setJobGrade(grade);
		} else if (bulk && e.key === "ArrowLeft") go(-1);
		else if (bulk && e.key === "ArrowRight") go(1);
	};

	const submit = async () => {
		const plan = planSaves(
			batch().map((j) => j.ID),
			batchGrade(),
			overrides(),
			reasons(),
		);
		setPending(true);
		const results = await Promise.allSettled(
			plan.map((p) => set.mutateAsync(p)),
		);
		setPending(false);
		const landed = plan.filter((_, i) => results[i]?.status === "fulfilled");
		const all = [...saved(), ...landed.map((p) => p.jobId)];
		setSaved(all);
		if (landed.length === plan.length) {
			const known = priors() ?? {};
			props.onDone(
				all.filter((id) => id in known),
				known,
			);
			return;
		}
		const landedIds = new Set(landed.map((p) => p.jobId));
		setBatch((b) => b.filter((j) => !landedIds.has(j.ID)));
		setIndex(0);
		setFailed(true);
	};

	const submitLabel = () => {
		if (pending()) return "Saving…";
		if (!bulk) return "Save grade";
		return failed()
			? `Retry ${batch().length}`
			: `Grade ${batch().length} jobs`;
	};

	return (
		<DialogContent onKeyDown={onKeyDown}>
			<div class="flex flex-col gap-5">
				<DialogHeader class="mb-0">
					<DialogTitle>
						{bulk ? `Grade ${jobs.length} jobs` : "Grade job"}
					</DialogTitle>
				</DialogHeader>

				<Show when={bulk}>
					<fieldset class="flex flex-col gap-2">
						<legend class="mb-2 text-xs font-medium text-muted">
							Grade for all
						</legend>
						<div class="grid grid-cols-3 gap-2">
							<For each={GRADES}>
								{(value) => (
									<Button
										type="button"
										variant={batchGrade() === value ? "default" : "outline"}
										aria-pressed={batchGrade() === value}
										onClick={() => setBatchGrade(value)}
									>
										{GRADE_LABEL[value]}
									</Button>
								)}
							</For>
						</div>
					</fieldset>
				</Show>

				<section
					aria-label="Current job"
					class={cn(
						"flex flex-col gap-4",
						bulk && "rounded-xl border border-border p-4",
					)}
				>
					<Show when={bulk}>
						<div class="flex items-center gap-3">
							<div class="flex flex-1 gap-1" aria-hidden="true">
								<For each={batch()}>
									{(j, i) => (
										<span
											class={cn(
												"h-1 flex-1 rounded-full",
												i() === index()
													? "bg-primary"
													: overrides()[j.ID] || reasons()[j.ID]?.length
														? "bg-accent-border"
														: "bg-border",
											)}
										/>
									)}
								</For>
							</div>
							<span class="font-mono text-xs tabular-nums text-faint">
								{index() + 1} / {batch().length}
							</span>
						</div>
					</Show>

					<div class="flex flex-col gap-1.5">
						<p class="text-base font-semibold text-foreground">{job().Title}</p>
						<div class="flex items-center gap-2 text-sm text-muted">
							<span class="truncate">{titleCase(job().CompanySlug)}</span>
							<Show when={job().Band || undefined}>
								{(band) => (
									<>
										<span class="text-faint">·</span>
										<span class="text-xs text-faint">scored</span>
										<BandChip band={band()} />
									</>
								)}
							</Show>
						</div>
					</div>

					<div class="flex flex-col gap-2">
						<div class="flex items-center justify-between">
							<p class="text-xs font-medium text-muted">
								{bulk ? "This job" : "Grade"}
							</p>
							<Show when={bulk}>
								<Show
									when={overrides()[job().ID]}
									fallback={
										<span class="text-xs text-faint">
											{batchGrade()
												? "Uses the grade for all"
												: "Pick a grade for all first"}
										</span>
									}
								>
									<button
										type="button"
										onClick={clearOverride}
										class="text-xs font-medium text-accent-text hover:underline"
									>
										Reset to grade for all
									</button>
								</Show>
							</Show>
						</div>
						<div class="grid grid-cols-3 gap-2">
							<For each={GRADES}>
								{(value, i) => (
									<Button
										type="button"
										size="sm"
										variant={
											effective(job().ID) !== value
												? "outline"
												: overrides()[job().ID] || !bulk
													? "default"
													: "secondary"
										}
										aria-pressed={effective(job().ID) === value}
										onClick={() => setJobGrade(value)}
										class="justify-between"
									>
										{GRADE_LABEL[value]}
										<kbd class="font-mono text-xs font-normal opacity-60">
											{i() + 1}
										</kbd>
									</Button>
								)}
							</For>
						</div>
						<Show when={effective(job().ID)}>
							{(g) => (
								<Show when={job().Band || undefined}>
									{(band) => (
										<p class="text-xs text-muted">
											Graded {GRADE_LABEL[g()]} · scored {BAND_LABEL[band()]}:{" "}
											{gradeDirection(g(), band())}
										</p>
									)}
								</Show>
							)}
						</Show>
					</div>

					<div class="flex flex-col gap-2" data-no-hotkeys>
						<p class="text-xs font-medium text-muted">
							Reasons <span class="text-faint">· optional</span>
						</p>
						<GradeChips
							selected={jobReasons()}
							onChange={(next) =>
								setReasons((r) => ({ ...r, [job().ID]: next }))
							}
						/>
					</div>

					<Show when={bulk}>
						<div class="flex justify-between">
							<Button
								variant="ghost"
								size="sm"
								disabled={index() === 0}
								onClick={() => go(-1)}
							>
								<Icon name="chevronLeft" size={14} />
								Previous
							</Button>
							<Button
								variant="ghost"
								size="sm"
								disabled={index() === batch().length - 1}
								onClick={() => go(1)}
							>
								Next
								<Icon name="chevronRight" size={14} />
							</Button>
						</div>
					</Show>
				</section>

				<Show when={failed()}>
					<p role="alert" class="text-sm text-destructive">
						{batch().length} couldn't be saved. Try again.
					</p>
				</Show>

				<DialogFooter class="mt-0 border-t border-border pt-4">
					<Button variant="outline" onClick={props.onCancel}>
						Cancel
					</Button>
					<Button
						disabled={!batchGrade() || pending() || (bulk && !priors())}
						onClick={submit}
					>
						{submitLabel()}
					</Button>
				</DialogFooter>
			</div>
		</DialogContent>
	);
}
