export function workStyle(
	daysInOffice: number | null | undefined,
): string | null {
	if (daysInOffice === null || daysInOffice === undefined) return null;
	if (daysInOffice === 0) return "Remote";
	if (daysInOffice >= 5) return "On-site";
	return `Hybrid · ${daysInOffice}d/wk`;
}
