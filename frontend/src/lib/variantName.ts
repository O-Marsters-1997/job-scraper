import type { AchievementVariant } from "../types/experience";

const NAME_WORDS = 6;

export function variantName(v: Pick<AchievementVariant, "text" | "label">) {
	const label = v.label.trim();
	if (label) return label;
	const words = v.text.trim().split(/\s+/);
	return words.length > NAME_WORDS
		? `${words.slice(0, NAME_WORDS).join(" ")}…`
		: words.join(" ");
}
