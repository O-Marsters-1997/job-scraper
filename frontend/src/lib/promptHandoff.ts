import type { PromptLab, PromptTry } from "@/types/promptLab";

export type HandoffInput = {
	brief: string;
	lab: PromptLab;
	texts: Record<string, string>;
	tries: PromptTry[];
};

export function tailoringPrompt(input: HandoffInput): string {
	const { lab, texts, tries } = input;
	const editable = lab.parts.map((p) => p.path);
	const out = [
		"I'm tuning the prompt my job app uses to tailor my CV. Help me improve it.",
		"",
		"## What I want",
		input.brief.trim() ||
			"(I haven't written a brief. Infer what to fix from the tries below.)",
		"",
		"## How the prompt is built",
		`The model receives a shared system prompt I can't change, then the documents below, then the instructions in ${lab.main}.`,
		`I can edit ${editable.join(" and ")}. ${lab.documents.map((d) => d.path).join(" and ")} come from the Draft and are read-only.`,
		"",
		"## Checks the app runs on every output",
		...lab.checks.map((c) => `- ${c}`),
		"",
		"## Files",
	];
	for (const f of [...lab.documents, ...lab.parts])
		out.push(`<file path="${f.path}">`, texts[f.path] ?? f.text, "</file>", "");
	out.push(`## Tries this session, newest first (${tries.length})`);
	for (const t of [...tries].sort((a, b) => b.n - a.n))
		out.push(
			`### Try ${t.n}${t.changed.length ? `, after editing ${t.changed.join(" and ")}` : ""}${t.steer ? `, steer "${t.steer}"` : ""}`,
			t.text,
			t.findings.length
				? t.findings.map((f) => `- ${f.check}: ${f.message}`).join("\n")
				: "- Passed every check.",
			"",
		);
	out.push(
		"## How to reply",
		`Return only the files you changed, each in full as <file path="NAME">…</file>, using ${editable.join(" or ")}. Then one line per change saying why. Keep edits small, and keep the output rules in ${lab.main}.`,
	);
	return out.join("\n");
}

export type ParsedReply = {
	changes: { path: string; from: string; to: string }[];
	skipped: string[];
	empty: boolean;
};

export function parseReply(
	reply: string,
	texts: Record<string, string>,
): ParsedReply {
	const found = [
		...reply.matchAll(/<file path="([^"]+)"[^>]*>([\s\S]*?)<\/file>/g),
	];
	const changes: ParsedReply["changes"] = [];
	const skipped: string[] = [];
	for (const [, path = "", body = ""] of found) {
		const from = texts[path];
		if (from === undefined) skipped.push(path);
		else if (from.trim() !== body.trim())
			changes.push({ path, from, to: body.trim() });
	}
	return { changes, skipped, empty: found.length === 0 };
}
