import type { Stance } from "@/types/scoringConfig";

export const STANCE_TONE: Record<Stance, string> = {
	nice: "border-accent-border bg-accent-subtle text-accent-text",
	ok: "border-accent-border bg-surface text-accent-text",
	avoid: "border-destructive/40 bg-surface text-destructive-strong",
	never: "border-destructive bg-destructive-subtle text-destructive-strong",
};

export type Level = 1 | 2 | 3 | 4 | 5;

export const LEVELS: {
	level: Level;
	stance: Stance | undefined;
	name: string;
	meaning: string;
	tone: string;
}[] = [
	{
		level: 1,
		stance: "never",
		name: "Never",
		meaning: "rules a job out",
		tone: "bg-destructive-strong text-primary-foreground",
	},
	{
		level: 2,
		stance: "avoid",
		name: "Not keen",
		meaning: "small penalty",
		tone: "bg-destructive-subtle text-destructive-strong ring-1 ring-destructive/30",
	},
	{
		level: 3,
		stance: undefined,
		name: "Don't mind",
		meaning: "no effect",
		tone: "bg-surface-muted text-foreground ring-1 ring-border-strong",
	},
	{
		level: 4,
		stance: "ok",
		name: "Keen",
		meaning: "some credit",
		tone: "bg-surface text-accent-text ring-1 ring-accent-border",
	},
	{
		level: 5,
		stance: "nice",
		name: "Love",
		meaning: "full credit",
		tone: "bg-primary/15 text-accent-text ring-1 ring-accent-border",
	},
];

export const levelOf = (stance: Stance | undefined): Level =>
	LEVELS.find((l) => l.stance === stance)?.level ?? 3;
