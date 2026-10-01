import { Link } from "@tanstack/solid-router";
import { createMemo, createSignal, For, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { selectedAchievementIds } from "@/lib/tailoring";
import { MissingAiKeyError } from "../../../api/tailoring";
import { useExperience } from "../../../hooks/useExperience";
import { type CVRef, useSuggestions } from "../../../hooks/useTailoring";
import type { Suggestion } from "../../../types/tailoring";

export function AchievementsStep(props: {
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
				<Card class="block p-5">
					<p class="text-sm text-foreground">
						Ranking Achievements against this job needs an OpenRouter key.{" "}
						<Link to="/settings/ai" class="text-accent-text underline">
							Add one in Settings, AI
						</Link>
						, which is also used to tailor your CV.
					</p>
				</Card>
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
