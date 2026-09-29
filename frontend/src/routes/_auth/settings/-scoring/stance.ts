export type Stance = "nice" | "avoid";

export const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	avoid: "border-destructive/40 bg-surface text-destructive-strong",
};
