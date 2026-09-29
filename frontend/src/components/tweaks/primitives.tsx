import { cva } from "class-variance-authority";
import type { JSX } from "solid-js";
import { For, Show } from "solid-js";
import { cn } from "@/lib/utils";

export const selectableButtonVariants = cva(
	"bg-transparent transition-[border-color,background]",
	{
		variants: {
			selected: {
				true: "border-primary bg-accent-subtle",
				false: "border-border hover:border-border-strong",
			},
		},
		defaultVariants: { selected: false },
	},
);

interface SectionProps {
	label: string;
	hint?: string;
	children: JSX.Element;
}

export function Section(props: SectionProps) {
	return (
		<div class="border-t border-border px-3.5 py-2.5">
			<p class="mb-2 flex items-baseline gap-1.5 text-2xs font-bold uppercase tracking-[0.08em] text-faint">
				{props.label}
				<Show when={props.hint}>
					<span class="text-2xs font-normal normal-case tracking-normal opacity-65">
						{props.hint}
					</span>
				</Show>
			</p>
			{props.children}
		</div>
	);
}

interface SegOption {
	value: string;
	label: string;
}

interface SegControlProps {
	options: SegOption[];
	value: string;
	onChange: (value: string) => void;
}

export function SegControl(props: SegControlProps) {
	return (
		<div class="flex overflow-hidden rounded-md border border-border">
			<For each={props.options}>
				{(opt, i) => (
					<button
						type="button"
						onClick={() => props.onChange(opt.value)}
						aria-pressed={props.value === opt.value}
						class={cn(
							"flex-1 py-1.5 text-xs font-medium transition-colors",
							i() < props.options.length - 1 && "border-r border-border",
							props.value === opt.value
								? "bg-foreground text-surface"
								: "text-muted hover:bg-background hover:text-foreground",
						)}
					>
						{opt.label}
					</button>
				)}
			</For>
		</div>
	);
}
