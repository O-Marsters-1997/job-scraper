export type SignalNote = { label: string; note: string };

export function buildReason(overall: string, signals: SignalNote[]): string {
	return [
		overall.trim(),
		...signals
			.filter((s) => s.note.trim())
			.map((s) => `${s.label} → ${s.note.trim()}`),
	]
		.filter(Boolean)
		.join("\n");
}
