import type { JSX } from "solid-js";
import { cn } from "@/lib/utils";
import type { BulletLean } from "@/types/tailoring";

const TONE = {
	fit: "text-status-offer",
	unclear: "text-status-interview",
	low: "text-destructive-strong",
} as const;

export const fitPercent = (score: number) => `${Math.round((score + 1) * 50)}%`;

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

export function Lean(props: { lean: BulletLean }) {
	const percent = () =>
		props.lean.state && props.lean.lean !== null
			? fitPercent(props.lean.lean)
			: null;
	return (
		<span
			data-testid="lean"
			role="img"
			aria-label={
				percent()
					? `Lean: ${props.lean.state} ${percent()}`
					: "Not in your Experience Bank"
			}
			class={cn(
				"pointer-events-none absolute top-0 left-[calc(100%+6pt)] whitespace-nowrap font-mono text-2xs",
				props.lean.state ? TONE[props.lean.state] : "text-faint",
			)}
		>
			{percent() ?? "—"}
		</span>
	);
}
