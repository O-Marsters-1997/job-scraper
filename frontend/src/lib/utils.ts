import { type ClassValue, clsx } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

// Register the project's custom font-size tokens (see @theme in styles.css) so
// tailwind-merge classifies e.g. `text-auth-action` as a font-size, not a
// text-color. Without this it treats unknown `text-*` as a colour and silently
// strips `text-primary-foreground` from buttons, leaving the label to inherit
// the dark foreground.
const twMerge = extendTailwindMerge({
	extend: {
		classGroups: {
			"font-size": [
				{
					text: [
						"2xs",
						"data",
						"auth-heading",
						"auth-subtext",
						"auth-action",
						"display",
						"display-prose",
					],
				},
			],
		},
	},
});

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

export function titleCase(slug: string): string {
	return slug
		.split(/[-_\s]+/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(" ");
}

export const plural = (n: number, word: string) =>
	`${n} ${word}${n === 1 ? "" : "s"}`;
