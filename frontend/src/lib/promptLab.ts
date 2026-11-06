import type { PromptLab } from "@/types/promptLab";

export type SentRow = {
	path: string;
	source: "From this Draft" | "Default" | "Instructions";
	text: string;
};

export const wordCount = (s: string) =>
	s.trim().split(/\s+/).filter(Boolean).length;

export function referencedPaths(text: string): string[] {
	return [...new Set(text.match(/[\w-]+\.md\b/g) ?? [])];
}

export function mainText(lab: PromptLab): string {
	return lab.parts.find((p) => p.path === lab.main)?.text ?? "";
}

export function missingReferences(lab: PromptLab): string[] {
	const known = new Set([
		...lab.parts.map((p) => p.path),
		...lab.documents.map((d) => d.path),
	]);
	return referencedPaths(mainText(lab)).filter(
		(p) => p !== lab.main && !known.has(p),
	);
}

export function sentRows(lab: PromptLab): SentRow[] {
	return [
		...lab.documents.map((d) => ({
			...d,
			source: "From this Draft" as const,
		})),
		...lab.parts
			.filter((p) => p.path !== lab.main)
			.map((p) => ({ ...p, source: "Default" as const })),
		{
			path: lab.main,
			source: "Instructions" as const,
			text: mainText(lab),
		},
	];
}

export function changedPaths(
	lab: PromptLab,
	edited: Record<string, string>,
): string[] {
	return lab.parts
		.filter((p) => edited[p.path] !== undefined && edited[p.path] !== p.text)
		.map((p) => p.path);
}

export function editedParts(
	lab: PromptLab,
	edited: Record<string, string>,
): Record<string, string> {
	return Object.fromEntries(
		changedPaths(lab, edited).map((path) => [path, edited[path] ?? ""]),
	);
}

export function baseProfile(lab: PromptLab): string {
	const text = lab.documents.find((d) => d.path === "profile.md")?.text ?? "";
	return text.replace(/^---\n[\s\S]*?\n---\n\n?/, "");
}

export function rawMessage(lab: PromptLab): string {
	const docs = sentRows(lab)
		.filter((r) => r.source !== "Instructions")
		.map((r) => `<document path="${r.path}">\n${r.text}\n</document>`)
		.join("\n");
	return `<documents>\n${docs}\n</documents>\n\n<instructions path="${lab.main}">\n${mainText(lab)}\n</instructions>`;
}
