import { For } from "solid-js";
import { cn } from "@/lib/utils";

const COUNTS = [1, 2, 3, 4];
const DEFAULT_COUNT = 3;

export function TechNeeded(props: {
	value: number | undefined;
	onChange: (count: number) => void;
}) {
	const selected = () => props.value ?? DEFAULT_COUNT;
	return (
		<div class="flex flex-col gap-1.5 rounded-md border border-border bg-surface-muted px-4 py-3">
			<div class="flex flex-wrap items-center gap-3 text-sm text-foreground">
				<span id="tech-needed-label">A job needs</span>
				<div
					role="radiogroup"
					aria-labelledby="tech-needed-label"
					class="inline-flex gap-0.5 rounded-md border border-border bg-surface p-0.5"
				>
					<For each={COUNTS}>
						{(n) => (
							<label
								class={cn(
									"min-w-7 cursor-pointer rounded-sm px-2 py-1 text-center font-mono text-xs font-medium tabular-nums transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-primary",
									selected() === n
										? "bg-primary/15 text-accent-text ring-1 ring-accent-border"
										: "text-faint hover:bg-surface-muted hover:text-foreground",
								)}
							>
								<input
									type="radio"
									class="sr-only"
									name="tech-needed"
									checked={selected() === n}
									onChange={() => props.onChange(n)}
								/>
								{n}
							</label>
						)}
					</For>
				</div>
				<span>techs earn about 90% of tech credit; more still helps.</span>
			</div>
			<p class="text-xs text-faint">
				A tech you rate 5 counts as one, a 4 counts as half.
			</p>
		</div>
	);
}
