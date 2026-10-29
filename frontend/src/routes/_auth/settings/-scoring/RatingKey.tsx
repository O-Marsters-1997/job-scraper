import { For } from "solid-js";
import { cn } from "@/lib/utils";
import { LEVELS } from "./stance";

export function RatingKey() {
	return (
		<div class="flex flex-col gap-3 border-b border-border pb-5">
			<p class="text-xs text-muted">
				Rate each option from 1 to 5. Anything you don't rate is a 3.
			</p>
			<dl class="flex flex-wrap gap-x-5 gap-y-2">
				<For each={LEVELS}>
					{(l) => (
						<div class="flex items-center gap-2 whitespace-nowrap">
							<span
								aria-hidden="true"
								class={cn(
									"min-w-6 rounded-sm px-1.5 py-0.5 text-center font-mono text-xs font-medium tabular-nums",
									l.tone,
								)}
							>
								{l.level}
							</span>
							<dt class="text-xs font-medium text-foreground">
								<span class="sr-only">{l.level}, </span>
								{l.name}
							</dt>
							<dd class="text-xs text-faint">{l.meaning}</dd>
						</div>
					)}
				</For>
			</dl>
		</div>
	);
}
