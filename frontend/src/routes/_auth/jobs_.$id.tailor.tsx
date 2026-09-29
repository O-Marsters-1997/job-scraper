import { createFileRoute, Link, useNavigate } from "@tanstack/solid-router";
import {
	createEffect,
	createMemo,
	createSignal,
	For,
	type JSX,
	Show,
} from "solid-js";
import { PageHeading } from "@/components/PageHeading";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import {
	allConfirmed,
	isSettled,
	selectedAchievementIds,
	toMappings,
} from "@/lib/tailoring";
import { MissingAiKeyError } from "../../api/tailoring";
import { GoogleWriteConsent } from "../../components/GoogleWriteConsent";
import { useCVTemplates } from "../../hooks/useCVTemplates";
import { useExperience } from "../../hooks/useExperience";
import { useGoogleStatus } from "../../hooks/useGoogle";
import {
	type CVRef,
	useCreateDraft,
	useDraft,
	useHeadings,
	useSaveHeadings,
	useSuggestions,
} from "../../hooks/useTailoring";
import type { CV } from "../../types/cv";
import type { Position } from "../../types/experience";
import type { CVHeading, Suggestion } from "../../types/tailoring";

export const Route = createFileRoute("/_auth/jobs_/$id/tailor")({
	component: TailorPage,
});

type Step = "cv" | "headings" | "achievements" | "generate";

const STEP_LABELS: { step: Step; label: string }[] = [
	{ step: "cv", label: "Base CV" },
	{ step: "headings", label: "Headings" },
	{ step: "achievements", label: "Achievements" },
	{ step: "generate", label: "Generate" },
];

function Stepper(props: { current: Step; skippedHeadings: boolean }) {
	return (
		<ol class="mb-6 flex flex-wrap gap-4 text-sm">
			<For each={STEP_LABELS}>
				{(s, i) => (
					<li
						classList={{
							"font-semibold text-foreground": props.current === s.step,
							"text-faint": props.current !== s.step,
							"line-through": s.step === "headings" && props.skippedHeadings,
						}}
					>
						{i() + 1}. {s.label}
					</li>
				)}
			</For>
		</ol>
	);
}

function Panel(props: { children: JSX.Element }) {
	return (
		<div class="rounded-xl border border-border bg-surface p-5">
			{props.children}
		</div>
	);
}

function TailorPage() {
	const params = Route.useParams();
	const navigate = useNavigate();
	const [cv, setCv] = createSignal<CVRef>();
	const [step, setStep] = createSignal<Step>("cv");
	const [skipped, setSkipped] = createSignal(false);
	const [achievementIds, setAchievementIds] = createSignal<string[]>([]);

	const pickCv = (ref: CVRef) => {
		setCv(ref);
		setSkipped(false);
		setStep("headings");
	};

	return (
		<div class="px-7 py-6 pb-16">
			<PageHeading
				title="Tailor CV"
				subtitle="Pick a base CV, confirm its roles, then choose the Achievements to feature"
			>
				<Button
					variant="outline"
					size="sm"
					onClick={() =>
						navigate({ to: "/jobs/$id", params: { id: params().id } })
					}
				>
					Back to job
				</Button>
			</PageHeading>
			<Stepper current={step()} skippedHeadings={skipped()} />

			<Show when={step() === "cv"}>
				<CvStep onPick={pickCv} />
			</Show>
			<Show when={step() === "headings" && cv()}>
				<HeadingsStep
					cv={cv}
					onDone={(wasSkipped) => {
						setSkipped(wasSkipped);
						setStep("achievements");
					}}
					onBack={() => setStep("cv")}
				/>
			</Show>
			<Show when={step() === "achievements" && cv()}>
				<AchievementsStep
					jobId={() => params().id}
					cv={cv}
					onBack={() => setStep(skipped() ? "cv" : "headings")}
					onContinue={(ids) => {
						setAchievementIds(ids);
						setStep("generate");
					}}
				/>
			</Show>
			<Show when={step() === "generate" && cv()}>
				<GenerateStep
					jobId={() => params().id}
					cv={cv}
					achievementIds={achievementIds}
					onBack={() => setStep("achievements")}
				/>
			</Show>
		</div>
	);
}

function CvStep(props: { onPick: (ref: CVRef) => void }) {
	const query = useCVTemplates();
	const visible = (cvs: CV[]) => cvs.filter((c) => c.Visible);
	return (
		<QueryBoundary query={query}>
			{(cvs) => (
				<Show
					when={visible(cvs()).length > 0}
					fallback={
						<Panel>
							<p class="text-sm text-muted">
								No CVs tracked yet.{" "}
								<Link to="/cv-templates" class="text-accent-text underline">
									Track a Google Doc
								</Link>{" "}
								first.
							</p>
						</Panel>
					}
				>
					<ul class="space-y-2">
						<For each={visible(cvs())}>
							{(c) => (
								<li>
									<button
										type="button"
										class="w-full rounded-xl border border-border bg-surface p-4 text-left transition-colors hover:border-primary"
										onClick={() =>
											props.onPick({ docId: c.DocID, tabId: c.TabID })
										}
									>
										<span class="block text-sm font-semibold text-foreground">
											{c.Title}
										</span>
										<span class="block text-xs text-muted">{c.SourceDoc}</span>
									</button>
								</li>
							)}
						</For>
					</ul>
				</Show>
			)}
		</QueryBoundary>
	);
}

function HeadingsStep(props: {
	cv: () => CVRef | undefined;
	onDone: (skipped: boolean) => void;
	onBack: () => void;
}) {
	const headings = useHeadings(() => props.cv());
	const positions = useExperience();
	const save = useSaveHeadings();
	const [chosen, setChosen] = createSignal<Record<string, string | null>>({});

	createEffect(() => {
		const data = headings.data;
		if (data && allConfirmed(data)) props.onDone(true);
	});

	const submit = (data: CVHeading[]) => {
		const ref = props.cv();
		if (!ref) return;
		save.mutate(
			{ ...ref, mappings: toMappings(data, chosen()) },
			{ onSuccess: () => props.onDone(false) },
		);
	};

	const valueFor = (h: CVHeading) =>
		h.text in chosen() ? (chosen()[h.text] ?? "") : (h.positionId ?? "");

	return (
		<QueryBoundary query={headings}>
			{(data) => (
				<Panel>
					<p class="mb-4 text-sm text-muted">
						Match each role heading in this CV to a Position. Choose None to
						leave a section untouched.
					</p>
					<div class="space-y-3">
						<For each={data()}>
							{(h) => (
								<label class="grid items-center gap-2 sm:grid-cols-[1fr_260px]">
									<span class="text-sm text-foreground">{h.text}</span>
									<select
										class="h-9 rounded-md border border-border bg-surface px-3 text-sm text-foreground"
										value={valueFor(h)}
										onChange={(e) =>
											setChosen((c) => ({
												...c,
												[h.text]: e.currentTarget.value || null,
											}))
										}
									>
										<option value="">None</option>
										<For each={positions.data ?? []}>
											{(p: Position) => (
												<option value={p.id}>
													{p.title}, {p.employer}
												</option>
											)}
										</For>
									</select>
								</label>
							)}
						</For>
					</div>
					<Show when={save.error}>
						<p class="mt-3 text-sm text-danger">Could not save the mapping.</p>
					</Show>
					<div class="mt-5 flex gap-2">
						<Button variant="outline" onClick={props.onBack}>
							Back
						</Button>
						<Button disabled={save.isPending} onClick={() => submit(data())}>
							Save and continue
						</Button>
					</div>
				</Panel>
			)}
		</QueryBoundary>
	);
}

function AchievementsStep(props: {
	jobId: () => string;
	cv: () => CVRef | undefined;
	onBack: () => void;
	onContinue: (achievementIds: string[]) => void;
}) {
	const suggestions = useSuggestions(
		() => props.jobId(),
		() => props.cv(),
	);
	const positions = useExperience();
	const [overrides, setOverrides] = createSignal<Record<string, boolean>>({});

	const isSelected = (s: Suggestion) =>
		overrides()[s.achievementId] ?? s.preselected;

	return (
		<Show
			when={!(suggestions.error instanceof MissingAiKeyError)}
			fallback={
				<Panel>
					<p class="text-sm text-foreground">
						Ranking Achievements against this job needs an OpenRouter key.{" "}
						<Link to="/settings/ai" class="text-accent-text underline">
							Add one in Settings, AI
						</Link>
						, which is also used to tailor your CV.
					</p>
				</Panel>
			}
		>
			<QueryBoundary query={suggestions} fallbackRows={4}>
				{(data) => {
					const grouped = createMemo(() =>
						(positions.data ?? [])
							.map((p) => ({
								position: p,
								items: data().filter((s) => s.positionId === p.id),
							}))
							.filter((g) => g.items.length > 0),
					);
					const selectedCount = () => data().filter(isSelected).length;
					return (
						<div class="space-y-4">
							<Show
								when={grouped().length > 0}
								fallback={
									<Panel>
										<p class="text-sm text-muted">
											Your Experience Bank has no Achievements yet.{" "}
											<Link to="/experience" class="text-accent-text underline">
												Add some
											</Link>
											.
										</p>
									</Panel>
								}
							>
								<For each={grouped()}>
									{(g) => (
										<Panel>
											<h2 class="mb-3 text-sm font-semibold text-foreground">
												{g.position.title}, {g.position.employer}
											</h2>
											<ul class="space-y-2">
												<For each={g.items}>
													{(s) => (
														<li>
															<label class="flex cursor-pointer items-start gap-3 text-sm">
																<input
																	type="checkbox"
																	class="mt-1"
																	checked={isSelected(s)}
																	onChange={(e) =>
																		setOverrides((o) => ({
																			...o,
																			[s.achievementId]:
																				e.currentTarget.checked,
																		}))
																	}
																/>
																<span class="flex-1 text-foreground">
																	{s.text}
																</span>
																<span class="font-mono text-xs tabular-nums text-faint">
																	{Math.round(s.score * 100)}
																</span>
															</label>
														</li>
													)}
												</For>
											</ul>
										</Panel>
									)}
								</For>
							</Show>
							<div class="flex items-center gap-3">
								<Button variant="outline" onClick={props.onBack}>
									Back
								</Button>
								<span class="text-xs text-muted">
									{selectedCount()} of {data().length} Achievements selected
								</span>
								<Button
									disabled={selectedCount() === 0}
									onClick={() =>
										props.onContinue(
											selectedAchievementIds(data(), overrides()),
										)
									}
								>
									Continue
								</Button>
							</div>
						</div>
					);
				}}
			</QueryBoundary>
		</Show>
	);
}

function GenerateStep(props: {
	jobId: () => string;
	cv: () => CVRef | undefined;
	achievementIds: () => string[];
	onBack: () => void;
}) {
	const google = useGoogleStatus();
	const create = useCreateDraft();
	const [draftId, setDraftId] = createSignal<string>();
	const draft = useDraft(draftId);

	const generate = () => {
		const ref = props.cv();
		if (!ref) return;
		create.mutate(
			{
				jobId: props.jobId(),
				...ref,
				achievementIds: props.achievementIds(),
			},
			{ onSuccess: (r) => setDraftId(r.id) },
		);
	};
	const retry = () => {
		setDraftId(undefined);
		generate();
	};

	return (
		<div class="space-y-4">
			<GoogleWriteConsent returnTo={`/jobs/${props.jobId()}/tailor`} />
			<Panel>
				<Show
					when={draftId()}
					fallback={
						<>
							<p class="mb-4 text-sm text-muted">
								FastTrack copies your CV into a new Google Doc and rewrites the
								bullets of the roles you mapped, using only the{" "}
								{props.achievementIds().length} Achievements you chose. Treat
								the result as a first draft.
							</p>
							<Show when={create.error}>
								<p class="mb-3 text-sm text-danger">
									Could not start the draft. Check your selection and try again.
								</p>
							</Show>
							<div class="flex gap-2">
								<Button variant="outline" onClick={props.onBack}>
									Back
								</Button>
								<Button
									disabled={
										create.isPending ||
										!google.data?.canWrite ||
										props.achievementIds().length === 0
									}
									onClick={generate}
								>
									Generate draft
								</Button>
							</div>
						</>
					}
				>
					<Show when={draft.data?.status === "ready"}>
						<p class="mb-3 text-sm text-foreground">Your draft is ready.</p>
						<div class="flex flex-wrap gap-2">
							<Link
								to="/tailoring/drafts/$id"
								params={{ id: draftId() ?? "" }}
								class="inline-flex h-8 items-center rounded-md bg-primary px-3 text-xs font-medium text-primary-foreground transition hover:bg-primary-hover"
							>
								Review draft
							</Link>
							<a
								href={draft.data?.draftDocUrl ?? undefined}
								target="_blank"
								rel="noopener noreferrer"
								class="inline-flex h-8 items-center rounded-md border border-border bg-surface px-3 text-xs font-medium text-foreground transition-colors hover:border-border-strong hover:bg-surface-muted"
							>
								Open in Google Docs
							</a>
						</div>
					</Show>
					<Show when={draft.data?.status === "failed"}>
						<p class="mb-3 text-sm text-danger">
							Generating the draft failed
							{draft.data?.lastError ? `: ${draft.data.lastError}` : "."}
						</p>
						<Button variant="outline" onClick={retry}>
							Try again
						</Button>
					</Show>
					<Show when={!draft.data || !isSettled(draft.data.status)}>
						<p class="text-sm text-muted" aria-live="polite">
							Writing your draft. This usually takes under a minute.
						</p>
					</Show>
				</Show>
			</Panel>
		</div>
	);
}
