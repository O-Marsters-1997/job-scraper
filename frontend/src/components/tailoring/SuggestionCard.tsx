import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { SlotSuggestion } from "../../hooks/useSuggestions";
import { ACTION_LABEL } from "./WandMenu";

export function SuggestionCard(props: {
	suggestion: SlotSuggestion;
	linesBefore: number;
	linesAfter: number | undefined;
	onAccept: () => void;
	onReject: () => void;
	onRetry: () => void;
}) {
	const s = () => props.suggestion;
	const delta = () =>
		props.linesAfter === undefined ? 0 : props.linesAfter - props.linesBefore;
	const changed = () => s().text !== s().before;
	return (
		<div
			data-testid="suggestion-card"
			class="rounded-xl bg-surface text-sm shadow-xl ring-1 ring-accent-border"
		>
			<div class="flex items-center gap-2 px-3.5 pt-3">
				<span class="grid size-5 place-items-center rounded-md bg-sidebar text-sidebar-active-foreground">
					<Icon name="wand" size={12} />
				</span>
				<span class="min-w-0 flex-1 truncate text-xs font-semibold text-foreground">
					{s().action === "ask" ? `"${s().prompt}"` : ACTION_LABEL[s().action]}
				</span>
				<span class="text-2xs text-faint">Haiku</span>
			</div>
			<div class="px-3.5 pt-2 pb-3.5">
				<Show
					when={s().done}
					fallback={
						<div class="flex items-center gap-2 text-xs text-muted">
							<span class="size-1.5 animate-pulse rounded-full bg-primary" />
							<span class="flex-1">Writing on the page…</span>
							<Button variant="ghost" size="sm" onClick={props.onReject}>
								Cancel
							</Button>
						</div>
					}
				>
					<Show
						when={s().error === undefined}
						fallback={
							<p role="alert" class="text-xs text-destructive-strong">
								{s().error}
							</p>
						}
					>
						<p class="text-xs text-muted">
							<Show
								when={changed()}
								fallback="Haiku found nothing to change here."
							>
								<Show when={delta() !== 0} fallback="Same number of lines.">
									<span
										data-testid="suggestion-effect"
										class={cn(
											"font-medium",
											delta() < 0
												? "text-status-offer"
												: "text-destructive-strong",
										)}
									>
										{delta() < 0 ? `Saves ${-delta()}` : `Adds ${delta()}`}{" "}
										{Math.abs(delta()) === 1 ? "line" : "lines"}
									</span>{" "}
									on the page.
								</Show>
							</Show>
						</p>
						<For each={s().findings}>
							{(f) => (
								<p class="mt-1.5 flex items-start gap-2 text-xs text-foreground">
									<span class="mt-1 size-1.5 shrink-0 rounded-full bg-status-interview" />
									{f.message}
								</p>
							)}
						</For>
					</Show>
					<div class="mt-3 flex items-center gap-1.5">
						<Show when={changed() && s().error === undefined}>
							<Button size="sm" onClick={props.onAccept}>
								<Icon name="check" size={14} />
								Accept
								<kbd class="ml-1 font-mono text-2xs opacity-70">⌘↵</kbd>
							</Button>
						</Show>
						<Button variant="ghost" size="sm" onClick={props.onReject}>
							{changed() && s().error === undefined ? "Reject" : "Close"}
						</Button>
						<Button
							variant="ghost"
							size="icon-sm"
							aria-label="Try again"
							class="ml-auto"
							onClick={props.onRetry}
						>
							<Icon name="rotateCcw" size={14} />
						</Button>
					</div>
				</Show>
			</div>
		</div>
	);
}
