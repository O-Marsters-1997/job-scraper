import { For, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import type { BulletLean } from "@/types/tailoring";
import { Tone } from "./Lean";

const LOW_LEAN = -0.2;

const percent = (lean: number) =>
	`${lean < 0 ? "−" : ""}${Math.round(Math.abs(lean) * 100)}%`;

const replacedState = (lean: number) =>
	lean > 0 ? "fit" : lean <= LOW_LEAN ? "low" : "unclear";

export function Insights(props: {
	lean: BulletLean[];
	text: (slotId: string) => string;
	onEdit: (slotId: string) => void;
}) {
	return (
		<section
			aria-label="Insights"
			class="rounded-xl border border-border bg-surface p-6 text-[14px]"
		>
			<h2 class="mb-3 text-sm font-semibold">Insights</h2>
			<Show
				when={props.lean.length > 0}
				fallback={<p class="text-muted">No bullets to read yet.</p>}
			>
				<ul class="divide-y divide-border">
					<For each={props.lean}>
						{(b) => (
							<li data-testid="insight" class="space-y-1 py-3">
								<p class="text-foreground">{props.text(b.slotId)}</p>
								<p class="font-mono text-xs">
									<Tone state={b.state}>{b.state ?? "—"}</Tone>
									<Show when={b.lean !== null}>
										<span class="text-faint"> {percent(b.lean ?? 0)}</span>
									</Show>
								</p>
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
	);
}
