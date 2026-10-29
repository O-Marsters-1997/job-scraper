import type { QuoteSpan } from "./quoteMarks";

export type TextPart = { kind: "p" | "h" | "li"; text: string };

const LOOKS_HTML = /<[a-z][\s\S]*?>/i;
const CRITERIA =
	/\s+(?=(?:Seniority level|Employment type|Job function|Industries):)/;
const BOLD_TAGS = new Set(["STRONG", "B"]);
const BULLET = /^\s*[-*•]\s+/;
const BLOCK_TAGS = new Set([
	"P",
	"DIV",
	"UL",
	"OL",
	"SECTION",
	"ARTICLE",
	"TABLE",
	"TR",
	"BLOCKQUOTE",
]);

function plainParts(text: string): TextPart[] {
	const parts: TextPart[] = [];
	for (const para of text.split(/\n\s*\n/)) {
		let current: TextPart | undefined;
		for (const line of para.split("\n")) {
			const bullet = BULLET.test(line);
			const clean = line.replace(BULLET, "").replace(/\s+/g, " ").trim();
			if (!clean) continue;
			if (current && !bullet) {
				current.text += ` ${clean}`;
				continue;
			}
			current = { kind: bullet ? "li" : "p", text: clean };
			parts.push(current);
		}
	}
	return parts;
}

function partKind(tag: string): TextPart["kind"] | null {
	if (/^H[1-6]$/.test(tag)) return "h";
	return tag === "LI" ? "li" : null;
}

function isBlock(tag: string): boolean {
	return partKind(tag) !== null || BLOCK_TAGS.has(tag);
}

function htmlParts(root: Node): TextPart[] {
	const parts: TextPart[] = [];
	let kind: TextPart["kind"] = "p";
	let buf = "";
	let plain = false;
	const flush = () => {
		const text = buf.replace(/\s+/g, " ").trim();
		if (text) {
			const heading = kind === "p" && !plain;
			parts.push({ kind: heading ? "h" : kind, text });
		}
		buf = "";
		plain = false;
	};
	const addText = (t: string | null, bold: boolean) => {
		buf += t ?? "";
		plain ||= !bold && /\S/.test(t ?? "");
	};
	const visit = (n: Node, bold: boolean) => {
		if (n.nodeType === Node.TEXT_NODE) {
			addText(n.textContent, bold);
			return;
		}
		if (n.nodeType !== Node.ELEMENT_NODE) return;
		const tag = (n as Element).tagName;
		if (tag === "BR") {
			flush();
			return;
		}
		const next = partKind(tag);
		const block = isBlock(tag);
		if (block) flush();
		const prev = kind;
		if (next) kind = next;
		const inBold = bold || BOLD_TAGS.has(tag);
		for (const child of n.childNodes) visit(child, inBold);
		if (block) flush();
		kind = prev;
	};
	visit(root, false);
	flush();
	return parts.flatMap(splitCriteria);
}

function splitCriteria(p: TextPart): TextPart[] {
	if (p.kind !== "p") return [p];
	return p.text.split(CRITERIA).map((text) => ({ kind: p.kind, text }));
}

export function htmlToParts(html: string): TextPart[] {
	const parse = (s: string) =>
		new DOMParser().parseFromString(s, "text/html").body;
	if (!LOOKS_HTML.test(html)) return plainParts(html);
	let body = parse(html);
	const text = body.textContent ?? "";
	if (LOOKS_HTML.test(text)) body = parse(text);
	return htmlParts(body);
}

const SEPARATOR = "\n\n";

export function joinParts(parts: TextPart[]): string {
	return parts.map((p) => p.text).join(SEPARATOR);
}

export function clipSpans(
	spans: QuoteSpan[],
	parts: TextPart[],
): { part: TextPart; spans: QuoteSpan[] }[] {
	let offset = 0;
	return parts.map((p) => {
		const start = offset;
		const end = start + p.text.length;
		offset = end + SEPARATOR.length;
		return {
			part: p,
			spans: spans
				.filter((s) => s.start < end && s.end > start)
				.map((s) => ({
					key: s.key,
					start: Math.max(s.start, start) - start,
					end: Math.min(s.end, end) - start,
				})),
		};
	});
}
