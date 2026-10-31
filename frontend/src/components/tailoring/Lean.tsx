import { Popover } from "@kobalte/core/popover";
import { type JSX, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { BulletLean, DraftInsights } from "@/types/tailoring";

const TONE = {
	fit: "text-status-offer-strong",
	unclear: "text-status-interview",
	low: "text-destructive-strong",
} as const;

export const fitPercent = (score: number) => `${Math.round((score + 1) * 50)}%`;

export type InsightLine = DraftInsights["lines"][number];

export function Tone(props: {
	state: keyof typeof TONE | null;
	children: JSX.Element;
}) {
	return (
		<span class={props.state ? TONE[props.state] : "text-faint"}>
			{props.children}
		</span>
	);
}

export function Lean(props: {
	lean: BulletLean;
	line: InsightLine | undefined;
	explaining: boolean;
	onExplain: () => void;
}) {
	const percent = () =>
		props.lean.state && props.lean.lean !== null
			? fitPercent(props.lean.lean)
			: null;
	return (
		<Popover placement="right-start">
			<Popover.Trigger
				type="button"
				data-testid="lean"
				aria-label={
					percent()
						? `Lean: ${props.lean.state} ${percent()}`
						: "Not in your Experience Bank"
				}
				class={cn(
					"absolute top-0 left-[calc(100%+6pt)] cursor-pointer whitespace-nowrap font-mono text-2xs hover:underline focus-visible:outline-2 focus-visible:outline-primary",
					props.lean.state ? TONE[props.lean.state] : "text-faint",
				)}
			>
				{percent() ?? "—"}
			</Popover.Trigger>
			<Popover.Portal>
				<Popover.Content
					data-tweaks-portal=""
					class="z-[60] w-72 space-y-2 rounded-xl border border-border bg-surface p-3 text-xs shadow-xl outline-none"
				>
					<p class="font-mono">
						<Tone state={props.lean.state}>{props.lean.state ?? "—"}</Tone>
						<Show when={props.lean.lean !== null}>
							<span class="text-faint">
								{" "}
								{fitPercent(props.lean.lean ?? 0)} fit
							</span>
						</Show>
					</p>
					<Show when={props.lean.state === null}>
						<p class="text-muted">Not in your Experience Bank.</p>
					</Show>
					<Show
						when={props.line}
						fallback={
							<Button
								variant="outline"
								size="sm"
								disabled={props.explaining}
								onClick={props.onExplain}
							>
								{props.explaining ? "Explaining…" : "Explain"}
							</Button>
						}
					>
						{(line) => (
							<p class="text-muted">
								Closest line: "{line().quote}" · {line().why}
							</p>
						)}
					</Show>
					<Show when={props.lean.replacedFrom}>
						{(old) => (
							<p class="text-muted">
								Replaced "{old()}"
								<Show when={props.lean.replacedLean !== null}>
									{" "}
									({fitPercent(props.lean.replacedLean ?? 0)})
								</Show>
							</p>
						)}
					</Show>
					<Show when={props.lean.closerMatch}>
						{(m) => (
							<p class="text-muted">
								Closer bank match ({fitPercent(m().lean)}): {m().text}
							</p>
						)}
					</Show>
				</Popover.Content>
			</Popover.Portal>
		</Popover>
	);
}
