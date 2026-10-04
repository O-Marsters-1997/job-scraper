export const MIN_REPLY_WINDOW_DAYS = 1;
export const MAX_REPLY_WINDOW_DAYS = 60;

export function addWorkingDays(date: Date, n: number): Date {
	const result = new Date(date);
	let remaining = n;
	while (remaining > 0) {
		result.setDate(result.getDate() + 1);
		const day = result.getDay();
		if (day !== 0 && day !== 6) remaining--;
	}
	while (result.getDay() === 0 || result.getDay() === 6) {
		result.setDate(result.getDate() + 1);
	}
	return result;
}

export type ReplyWindowInput =
	| { days: number | null; error?: undefined }
	| { days?: undefined; error: string };

export function parseReplyWindow(text: string): ReplyWindowInput {
	const trimmed = text.trim();
	if (trimmed === "") return { days: null };
	const days = Number(trimmed);
	if (
		!Number.isInteger(days) ||
		days < MIN_REPLY_WINDOW_DAYS ||
		days > MAX_REPLY_WINDOW_DAYS
	) {
		return {
			error: `Enter a whole number from ${MIN_REPLY_WINDOW_DAYS} to ${MAX_REPLY_WINDOW_DAYS}.`,
		};
	}
	return { days };
}
