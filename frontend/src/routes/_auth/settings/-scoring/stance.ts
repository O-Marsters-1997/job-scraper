import type { Stance } from "@/types/scoringConfig";

export const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	ok: "border-accent-border bg-surface text-accent-text",
	avoid: "border-destructive/40 bg-surface text-destructive-strong",
	block: "border-destructive bg-destructive-subtle text-destructive-strong",
};

export const STANCE_LABEL: Record<Stance, string> = {
	nice: "most want",
	ok: "happy with",
	avoid: "avoid",
	block: "never",
};
