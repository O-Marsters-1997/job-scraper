import { format } from "date-fns";

export function chaseDateKey(chaseBy: string): string {
	return chaseBy.slice(0, 10);
}

export function isChaseOverdue(chaseBy: string, now: Date): boolean {
	return chaseDateKey(chaseBy) < format(now, "yyyy-MM-dd");
}

export function formatChaseDate(chaseBy: string): string {
	const [year, month, day] = chaseDateKey(chaseBy).split("-").map(Number);
	return new Date(year!, month! - 1, day!).toLocaleDateString("en-GB", {
		day: "numeric",
		month: "short",
		year: "numeric",
	});
}
