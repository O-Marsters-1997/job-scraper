import { createSignal, For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { BulletLean, DraftInsights } from "@/types/tailoring";
import { JobText, type JobTextBlock } from "./JobText";
import { Tone } from "./Lean";

const LOW_LEAN = -0.2;

const percent = (lean: number) =>
	`${lean < 0 ? "−" : ""}${Math.round(Math.abs(lean) * 100)}%`;

const replacedState = (lean: number) =>
	lean > 0 ? "fit" : lean <= LOW_LEAN ? "low" : "unclear";

const SOURCE_LABEL = { jd: "JD", note: "Note", comment: "Comment" } as const;

export function Insights(props: {
	lean: BulletLean[];
	insights: DraftInsights | undefined;
	explaining: boolean;
	explainError: string | undefined;
	jobText: JobTextBlock[];
	text: (slotId: string) => string;
	onEdit: (slotId: string) => void;
	onExplain: () => void;
}) {
	const explained = () => (props.insights?.lines.length ?? 0) > 0;
	const lineFor = (slotId: string) =>
		props.insights?.lines.find((l) => l.slotId === slotId);
	const [selected, setSelected] = createSignal<string>();
	const toggle = (slotId: string) =>
		setSelected((s) => (s === slotId ? undefined : slotId));
	const marks = () =>
		props.lean.flatMap((b, i) => {
			const line = lineFor(b.slotId);
			return line?.anchored
				? [
						{
							slotId: b.slotId,
							n: i + 1,
							quote: line.quote,
							source: line.source,
						},
					]
				: [];
		});
	return (
		<div class="grid grid-cols-1 items-start lg:grid-cols-2 gap-4">
			<section
				aria-label="Insights"
				class="rounded-xl border border-border bg-surface p-6 text-[14px]"
			>
				<div class="mb-3 flex items-center gap-3">
					<h2 class="text-sm font-semibold">Insights</h2>
					<Show when={explained() && props.insights?.stale}>
						<span class="text-xs text-status-interview">out of date</span>
					</Show>
					<Button
						class="ml-auto"
						variant="outline"
						size="sm"
						disabled={props.explaining}
						onClick={props.onExplain}
					>
						{props.explaining
							? "Explaining…"
							: explained()
								? "Refresh"
								: "Explain"}
					</Button>
				</div>
				<Show when={props.explainError}>
					{(message) => (
						<p role="alert" class="mb-3 text-destructive-strong">
							{message()}
						</p>
					)}
				</Show>
				<Show
					when={props.lean.length > 0}
					fallback={<p class="text-muted">No bullets to read yet.</p>}
				>
					<ul class="divide-y divide-border">
						<For each={props.lean}>
							{(b, i) => (
								<li
									data-testid="insight"
									class={cn(
										"space-y-1 py-3",
										selected() === b.slotId && "bg-accent-subtle",
									)}
								>
									<button
										type="button"
										class="flex w-full gap-2 text-left text-foreground"
										aria-pressed={selected() === b.slotId}
										onClick={() => toggle(b.slotId)}
									>
										<span class="w-4 shrink-0 font-mono text-2xs leading-5 text-accent-text">
											{i() + 1}
										</span>
										<span>{props.text(b.slotId)}</span>
									</button>
									<p class="font-mono text-xs">
										<Tone state={b.state}>{b.state ?? "—"}</Tone>
										<Show when={b.lean !== null}>
											<span class="text-faint"> {percent(b.lean ?? 0)}</span>
										</Show>
									</p>
									<Show when={lineFor(b.slotId)}>
										{(line) => (
											<p data-testid="explanation" class="text-muted">
												<span class="font-mono text-xs text-faint">
													{SOURCE_LABEL[line().source]}:
												</span>{" "}
												"{line().quote}" · {line().why}{" "}
												<Show when={!line().anchored}>
													<span class="rounded border border-status-interview px-1 font-mono text-2xs text-status-interview">
														no anchor
													</span>
												</Show>
											</p>
										)}
									</Show>
									<Show when={b.replacedFrom}>
										{(old) => (
											<p class="text-muted">
												replaced "{old()}"
												<Show when={b.replacedLean !== null}>
													{" "}
													(
													<Tone state={replacedState(b.replacedLean ?? 0)}>
														{percent(b.replacedLean ?? 0)}{" "}
														{replacedState(b.replacedLean ?? 0)}
													</Tone>
													)
												</Show>
											</p>
										)}
									</Show>
									<Show when={b.closerMatch}>
										{(m) => (
											<p class="text-muted">
												Weaker than a bank match ({percent(m().lean)} vs{" "}
												{percent(b.lean ?? 0)}): {m().text}{" "}
												<Button
													variant="link"
													size="sm"
													onClick={() => props.onEdit(b.slotId)}
												>
													Edit in document →
												</Button>
											</p>
										)}
									</Show>
								</li>
							)}
						</For>
					</ul>
				</Show>
			</section>
			<JobText
				blocks={props.jobText}
				marks={marks()}
				selected={selected()}
				onSelect={toggle}
			/>
		</div>
	);
}
