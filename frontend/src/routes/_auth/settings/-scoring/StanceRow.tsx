import { For } from "solid-js";
import { cn } from "@/lib/utils";
import type { Stance } from "@/types/scoringConfig";
import type { ScoringOption } from "@/types/scoringOptions";
import { LEVELS, levelOf } from "./stance";

export function StanceRow(props: {
	option: ScoringOption;
	stance: Stance | undefined;
	allowed: Stance[];
	onChange: (stance: Stance | undefined) => void;
}) {
	const value = () => levelOf(props.stance);
	return (
		<div class="flex flex-col gap-2 border-b border-border py-3 last:border-b-0 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
			<p
				class={cn(
					"min-w-0 text-sm text-foreground",
					props.stance === "never" &&
						"text-muted line-through decoration-destructive",
				)}
			>
				{props.option.label}
			</p>
			<div
				role="radiogroup"
				aria-label={props.option.label}
				class="inline-flex shrink-0 gap-0.5 rounded-md border border-border bg-surface p-0.5"
			>
				<For each={LEVELS}>
					{(l) => {
						const disabled = () =>
							l.stance !== undefined && !props.allowed.includes(l.stance);
						return (
							<label
								title={l.name}
								class={cn(
									"min-w-7 rounded-sm px-2 py-1 text-center font-mono text-xs font-medium tabular-nums transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary",
									disabled()
										? "cursor-not-allowed text-faint opacity-40"
										: "cursor-pointer",
									value() === l.level
										? l.tone
										: "text-faint hover:bg-surface-muted hover:text-foreground",
								)}
							>
								<input
									type="radio"
									class="sr-only"
									name={props.option.id}
									disabled={disabled()}
									checked={value() === l.level}
									onChange={() => props.onChange(l.stance)}
								/>
								{l.level}
								<span class="sr-only">, {l.name}</span>
							</label>
						);
					}}
				</For>
			</div>
		</div>
	);
}
