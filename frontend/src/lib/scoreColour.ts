/** Map a 0–100 score to a band hex. Bands: ≥80 strong green, ≥65 lighter green, ≥50 amber, <50 red. */
export function scoreColour(n: number): string {
	if (n >= 80) return "#059669"; // strong green (mirrors status-offer)
	if (n >= 65) return "#10b981"; // lighter green
	if (n >= 50) return "#d97706"; // amber (mirrors status-interview)
	return "#dc2626"; // red (mirrors status-rejected)
}
