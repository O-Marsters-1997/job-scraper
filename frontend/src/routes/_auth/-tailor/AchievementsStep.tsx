import { Link } from "@tanstack/solid-router";
import { createMemo, createSignal, For, type Setter, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
	canExplain,
	moveSuggestion,
	orderSuggestions,
	selectedAchievementIds,
} from "@/lib/tailoring";
import { MissingAiKeyError } from "../../../api/tailoring";
import { useExperience } from "../../../hooks/useExperience";
import {
	type CVRef,
	useExplainAchievement,
	useSuggestions,
} from "../../../hooks/useTailoring";
import type { Suggestion } from "../../../types/tailoring";

function MissingKeyPrompt() {
	return (
		<Card class="block p-5">
			<p class="text-sm text-foreground">
				Ranking Achievements against this job needs an OpenRouter key.{" "}
				<Link to="/settings/ai" class="text-accent-text underline">
					Add one in Settings, AI
				</Link>
				, which is also used to tailor your CV.
			</p>
		</Card>
	);
}

function ExplainWhy(props: { jobId: () => string; achievementId: string }) {
	const explain = useExplainAchievement(() => props.jobId());
	return (
		<div class="mt-1 text-xs">
			<Show
				when={explain.data}
				fallback={
					<button
						type="button"
						class="text-accent-text underline disabled:opacity-50"
						disabled={explain.isPending}
						onClick={() => explain.mutate(props.achievementId)}
					>
						{explain.isPending ? "Thinking" : "Why?"}
					</button>
				}
			>
				{(e) => (
					<p class="text-muted">
						<span class="font-medium">A guess, not Jev's reasoning:</span>{" "}
						{e().text}
					</p>
				)}
			</Show>
			<Show when={explain.error}>
				{(err) => (
					<Show
						when={err() instanceof MissingAiKeyError}
						fallback={
							<p role="alert" class="text-danger">
								Could not explain this bullet. Try again.
							</p>
						}
					>
						<MissingKeyPrompt />
					</Show>
				)}
			</Show>
		</div>
	);
}

export function AchievementsStep(props: {
	jobId: () => string;
	cv: () => CVRef | undefined;
	overrides: () => Record<string, boolean>;
	setOverrides: Setter<Record<string, boolean>>;
	order: () => string[];
	setOrder: Setter<string[]>;
	onBack: () => void;
	onContinue: (achievementIds: string[]) => void;
}) {
	const suggestions = useSuggestions(
		() => props.jobId(),
		() => props.cv(),
	);
	const positions = useExperience();

	const [dragged, setDragged] = createSignal<string>();

	const isSelected = (s: Suggestion) =>
		props.overrides()[s.achievementId] ?? s.preselected;

	return (
		<Show
			when={!(suggestions.error instanceof MissingAiKeyError)}
			fallback={<MissingKeyPrompt />}
		>
			<QueryBoundary query={suggestions} fallbackRows={4}>
				{(data) => {
					const ordered = createMemo(() =>
						orderSuggestions(data(), props.order()),
					);
					const move = (id: string, to: number) =>
						props.setOrder(moveSuggestion(ordered(), id, to));
					const grouped = createMemo(() =>
						(positions.data ?? [])
							.map((p) => ({
								position: p,
								items: ordered().filter((s) => s.positionId === p.id),
							}))
							.filter((g) => g.items.length > 0),
					);
					const selectedCount = () => data().filter(isSelected).length;
					const isLastPicked = (s: Suggestion, items: Suggestion[]) =>
						isSelected(s) && items.filter(isSelected).length === 1;
					return (
						<div class="space-y-4">
							<Show
								when={grouped().length > 0}
								fallback={
									<Card class="block p-5">
										<p class="text-sm text-muted">
											Your Experience Bank has no Achievements yet.{" "}
											<Link to="/experience" class="text-accent-text underline">
												Add some
											</Link>
											.
										</p>
									</Card>
								}
							>
								<For each={grouped()}>
									{(g) => (
										<Card class="block p-5">
											<h2 class="mb-3 text-sm font-semibold text-foreground">
												{g.position.title}, {g.position.employer}
											</h2>
											<ul class="space-y-2">
												<For each={g.items}>
													{(s, i) => (
														<li
															class="flex items-start gap-2 rounded-md p-1.5 text-sm"
															classList={{
																"opacity-50": !isSelected(s),
																"bg-surface-muted":
																	dragged() === s.achievementId,
															}}
															draggable={true}
															onDragStart={() => setDragged(s.achievementId)}
															onDragEnd={() => setDragged(undefined)}
															onDragOver={(e) => e.preventDefault()}
															onDrop={() => {
																const from = dragged();
																if (from && from !== s.achievementId)
																	move(from, i());
															}}
														>
															<span
																aria-hidden="true"
																class="cursor-grab select-none text-faint"
															>
																⠿
															</span>
															<input
																type="checkbox"
																class="mt-1"
																aria-label={s.text}
																checked={isSelected(s)}
																disabled={isLastPicked(s, g.items)}
																onChange={(e) =>
																	props.setOverrides((o) => ({
																		...o,
																		[s.achievementId]: e.currentTarget.checked,
																	}))
																}
															/>
															<div class="flex-1">
																<span class="text-foreground">{s.text}</span>
																<Show when={s.state !== "fit"}>
																	<Badge variant="outline">
																		{s.state === "low" ? "Low fit" : "Unclear"}
																	</Badge>
																</Show>
																<Show when={canExplain(s)}>
																	<ExplainWhy
																		jobId={props.jobId}
																		achievementId={s.achievementId}
																	/>
																</Show>
															</div>
															<span class="font-mono text-xs tabular-nums text-faint">
																{Math.round((s.score + 1) * 50)}%
															</span>
															<button
																type="button"
																aria-label="Move up"
																disabled={i() === 0}
																onClick={() => move(s.achievementId, i() - 1)}
																class="rounded px-1 text-muted hover:bg-surface-muted disabled:opacity-30"
															>
																↑
															</button>
															<button
																type="button"
																aria-label="Move down"
																disabled={i() === g.items.length - 1}
																onClick={() => move(s.achievementId, i() + 1)}
																class="rounded px-1 text-muted hover:bg-surface-muted disabled:opacity-30"
															>
																↓
															</button>
														</li>
													)}
												</For>
											</ul>
										</Card>
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
											selectedAchievementIds(
												data(),
												props.overrides(),
												props.order(),
											),
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
