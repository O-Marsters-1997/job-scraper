import { cn } from "@/lib/utils";
import type { BulletLean } from "@/types/tailoring";

const TONE = {
	fit: "text-status-offer",
	unclear: "text-status-interview",
	low: "text-destructive-strong",
} as const;

export function Lean(props: { lean: BulletLean }) {
	return (
		<span
			data-testid="lean"
			role="img"
			aria-label={
				props.lean.state
					? `Lean: ${props.lean.state}`
					: "Not in your Experience Bank"
			}
			class={cn(
				"pointer-events-none absolute top-0 left-[calc(100%+6pt)] font-mono text-2xs",
				props.lean.state ? TONE[props.lean.state] : "text-faint",
			)}
		>
			{props.lean.state ?? "—"}
		</span>
	);
}
