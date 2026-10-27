const BLOCK_END = /<\/(p|li|div|h[1-6]|ul|ol)>|<br\s*\/?>/gi;
const LOOKS_HTML = /<[a-z][\s\S]*?>/i;

function strip(html: string): string {
	const doc = new DOMParser().parseFromString(
		html.replace(BLOCK_END, "$&\n"),
		"text/html",
	);
	return doc.body.textContent ?? "";
}

export function htmlToText(html: string): string {
	let text = strip(html);
	if (LOOKS_HTML.test(text)) text = strip(text);
	return text
		.replace(/[ \t]+\n/g, "\n")
		.replace(/\n{3,}/g, "\n\n")
		.trim();
}
