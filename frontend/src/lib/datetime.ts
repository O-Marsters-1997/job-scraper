const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function formatDate(iso: string): string {
	return new Date(iso).toLocaleDateString("en-GB", {
		day: "numeric",
		month: "short",
		year: "numeric",
	});
}

export function formatRelative(iso: string): string {
	const diffMs = new Date(iso).getTime() - Date.now();
	const diffSec = Math.round(diffMs / 1000);
	const diffMin = Math.round(diffSec / 60);
	const diffHr = Math.round(diffMin / 60);
	const diffDay = Math.round(diffHr / 24);
	const diffWk = Math.round(diffDay / 7);
	const diffMo = Math.round(diffDay / 30);
	const abs = (n: number) => Math.abs(n);

	if (abs(diffSec) < 60) return "just now";
	if (abs(diffMin) < 60) return rtf.format(diffMin, "minute");
	if (abs(diffHr) < 24) return rtf.format(diffHr, "hour");
	if (abs(diffDay) < 7) return rtf.format(diffDay, "day");
	if (abs(diffWk) < 5) return rtf.format(diffWk, "week");
	return rtf.format(diffMo, "month");
}
