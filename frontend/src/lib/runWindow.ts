import type { RunWindow } from "@/types/sourceTarget";

export const DEFAULT_RUN_WINDOW: RunWindow = {
	interval_minutes: 60,
	weekdays: [1, 2, 3, 4, 5],
	start: "08:00",
	end: "18:00",
	timezone: "Europe/London",
};

export const INTERVAL_OPTIONS = [
	{ value: "60", label: "Hourly" },
	{ value: "180", label: "Every 3 hours" },
	{ value: "360", label: "Every 6 hours" },
	{ value: "1440", label: "Daily" },
];

export const WEEKDAY_CHIPS = [
	{ day: 1, label: "Mon" },
	{ day: 2, label: "Tue" },
	{ day: 3, label: "Wed" },
	{ day: 4, label: "Thu" },
	{ day: 5, label: "Fri" },
	{ day: 6, label: "Sat" },
	{ day: 0, label: "Sun" },
];

export interface WindowDraft {
	automatic: boolean;
	interval: number;
	weekdays: number[];
	start: string;
	end: string;
	timezone: string;
}

export function draftFrom(window: RunWindow, automatic?: boolean): WindowDraft {
	return {
		automatic: automatic ?? window.interval_minutes !== null,
		interval: window.interval_minutes ?? DEFAULT_RUN_WINDOW.interval_minutes!,
		weekdays: window.weekdays.length
			? window.weekdays
			: DEFAULT_RUN_WINDOW.weekdays,
		start: window.start,
		end: window.end,
		timezone: window.timezone,
	};
}

export function draftError(draft: WindowDraft): string | null {
	if (!draft.automatic) return null;
	if (draft.weekdays.length === 0) return "Pick at least one day.";
	if (!draft.start || !draft.end || draft.start >= draft.end) {
		return "The end time must be after the start time.";
	}
	return null;
}

export function toRunWindow(draft: WindowDraft): RunWindow | null {
	if (!draft.automatic) return null;
	return {
		interval_minutes: draft.interval,
		weekdays: draft.weekdays,
		start: draft.start,
		end: draft.end,
		timezone: draft.timezone,
	};
}

export function describeNextRun(
	nextRunAt: string | null,
	now = new Date(),
): string {
	if (!nextRunAt) return "Manual";
	const next = new Date(nextRunAt);
	const time = next.toLocaleTimeString("en-GB", {
		hour: "2-digit",
		minute: "2-digit",
	});
	if (next.toDateString() === now.toDateString()) return `Next run ${time}`;
	const day = next.toLocaleDateString("en-GB", { weekday: "short" });
	return `Next run ${day} ${time}`;
}
