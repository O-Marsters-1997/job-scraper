export function scoreColour(n: number): string {
	if (n >= 80) return "#059669";
	if (n >= 65) return "#10b981";
	if (n >= 50) return "#d97706";
	return "#dc2626";
}
