import type { SlotEdit } from "@/types/tailoring";

export const ADDED_PREFIX = "new-";

export const isAddedSlot = (slotId: string) => slotId.startsWith(ADDED_PREFIX);

export function slotsToSave(
	ids: string[],
	texts: Record<string, string>,
	saved: Record<string, string>,
	removed: string[],
): SlotEdit[] {
	const out: SlotEdit[] = [];
	let previous = "";
	for (const slotId of ids) {
		const text = (texts[slotId] ?? "").trim();
		if (!isAddedSlot(slotId)) {
			if (text !== "") out.push({ slotId, text });
			previous = slotId;
		} else if (text !== "") {
			out.push({ slotId, text, after: previous });
			previous = slotId;
		} else if (slotId in saved) {
			out.push({ slotId, text: "", remove: true });
		}
	}
	for (const slotId of removed)
		if (slotId in saved) out.push({ slotId, text: "", remove: true });
	return out;
}
