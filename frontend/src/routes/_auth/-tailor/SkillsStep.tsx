import { type Accessor, createMemo, For, type Setter, Show } from "solid-js";
import { MultiCombobox } from "@/components/MultiCombobox";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	candidatesFor,
	placeUnplaced,
	preselectedPicks,
	retarget,
	setLinePicks,
} from "@/lib/tailoring";
import { MissingAiKeyError } from "../../../api/tailoring";
import { type CVRef, useSkillSuggestions } from "../../../hooks/useTailoring";
import type {
	SkillLineSuggestion,
	SkillPick,
	SkillSuggestions,
} from "../../../types/tailoring";

const selectClass =
	"h-8 rounded-md border border-border bg-surface px-2 text-sm text-foreground";

function lineName(line: SkillLineSuggestion) {
	return line.label || "Skills";
}

function SkillsEditor(props: {
	data: SkillSuggestions;
	picks: SkillPick[];
	setPicks: (picks: SkillPick[]) => void;
}) {
	const pickOf = (id: string) => props.picks.find((p) => p.bankSkillId === id);
	return (
		<div class="space-y-4">
			<For each={props.data.lines}>
				{(line, i) => {
					const candidates = createMemo(() =>
						candidatesFor(props.data, i(), props.picks),
					);
					const linePicks = createMemo(() =>
						props.picks.filter((p) => p.line === i()),
					);
					return (
						<Card class="block p-5">
							<h2 class="mb-1 text-sm font-semibold text-foreground">
								{lineName(line)}
							</h2>
							<p class="mb-3 text-xs text-muted">
								Now: {line.base.map((b) => b.text).join(", ")}
							</p>
							<Show
								when={candidates().length > 0}
								fallback={
									<p class="text-sm text-muted">
										No Bank Skills to offer for this line.
									</p>
								}
							>
								<MultiCombobox
									label={`Swap into ${lineName(line)}`}
									hint="Fit skills are preselected. Each replaces one item already on your CV."
									options={candidates().map((c) => ({
										id: c.bankSkillId,
										label: c.state === "fit" ? `${c.name} (fit)` : c.name,
									}))}
									value={linePicks().map((p) => p.bankSkillId)}
									onChange={(ids) =>
										props.setPicks(
											setLinePicks(props.data, props.picks, i(), ids),
										)
									}
									placeholder="Add a skill…"
									chipClass="border-border bg-surface-muted text-foreground"
								/>
								<ul class="mt-3 flex flex-col gap-2">
									<For each={linePicks()}>
										{(p) => (
											<li class="flex flex-wrap items-center gap-2 text-sm">
												<span class="font-medium text-foreground">
													{candidates().find(
														(c) => c.bankSkillId === p.bankSkillId,
													)?.name ?? p.bankSkillId}
												</span>
												<span class="text-muted">replaces</span>
												<select
													class={selectClass}
													aria-label={`Item replaced by ${candidates().find((c) => c.bankSkillId === p.bankSkillId)?.name ?? ""}`}
													value={p.replaces}
													onChange={(e) =>
														props.setPicks(
															retarget(
																props.picks,
																p.bankSkillId,
																e.currentTarget.value,
															),
														)
													}
												>
													<For each={line.base}>
														{(b) => <option value={b.text}>{b.text}</option>}
													</For>
												</select>
											</li>
										)}
									</For>
								</ul>
							</Show>
						</Card>
					);
				}}
			</For>
			<Show when={props.data.unplaced.length > 0}>
				<Card class="block p-5">
					<h2 class="mb-1 text-sm font-semibold text-foreground">
						No matching line
					</h2>
					<p class="mb-3 text-xs text-muted">
						These Bank Skills have a category that matches none of your CV's
						lines. Pick a line to swap one in.
					</p>
					<ul class="flex flex-col gap-2">
						<For each={props.data.unplaced}>
							{(c) => (
								<li class="flex flex-wrap items-center gap-2 text-sm">
									<span class="font-medium text-foreground">{c.name}</span>
									<select
										class={selectClass}
										aria-label={`Line for ${c.name}`}
										value={pickOf(c.bankSkillId)?.line.toString() ?? ""}
										onChange={(e) => {
											const v = e.currentTarget.value;
											props.setPicks(
												placeUnplaced(
													props.data,
													props.picks,
													c.bankSkillId,
													v === "" ? undefined : Number(v),
												),
											);
										}}
									>
										<option value="">Don't swap in</option>
										<For each={props.data.lines}>
											{(l, i) => <option value={i()}>{lineName(l)}</option>}
										</For>
									</select>
								</li>
							)}
						</For>
					</ul>
				</Card>
			</Show>
		</div>
	);
}

export function SkillsStep(props: {
	jobId: () => string;
	cv: () => CVRef | undefined;
	picks: Accessor<SkillPick[] | undefined>;
	setPicks: Setter<SkillPick[] | undefined>;
	onBack: () => void;
	onContinue: () => void;
}) {
	const suggestions = useSkillSuggestions(
		() => props.jobId(),
		() => props.cv(),
	);
	return (
		<Show
			when={!(suggestions.error instanceof MissingAiKeyError)}
			fallback={
				<Card class="block p-5">
					<p class="text-sm text-foreground">
						Matching skills against this job needs an OpenRouter key. Add one in
						Settings, AI.
					</p>
				</Card>
			}
		>
			<QueryBoundary query={suggestions} fallbackRows={3}>
				{(data) => {
					const picks = () => props.picks() ?? preselectedPicks(data());
					const empty = () =>
						data().lines.every((l) => l.candidates.length === 0) &&
						data().unplaced.length === 0;
					return (
						<div class="space-y-4">
							<Show
								when={!empty()}
								fallback={
									<Card class="block p-5">
										<p class="text-sm text-muted">
											Nothing in your Bank Skills is missing from this CV.
										</p>
									</Card>
								}
							>
								<SkillsEditor
									data={data()}
									picks={picks()}
									setPicks={props.setPicks}
								/>
							</Show>
							<div class="flex items-center gap-3">
								<Button variant="outline" onClick={props.onBack}>
									Back
								</Button>
								<span class="text-xs text-muted">
									{picks().length} skill{picks().length === 1 ? "" : "s"} to
									swap in
								</span>
								<Button
									onClick={() => {
										props.setPicks(picks());
										props.onContinue();
									}}
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
