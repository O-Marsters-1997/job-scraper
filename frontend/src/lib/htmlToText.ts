const NAMED: Record<string, string> = {
	amp: "&",
	lt: "<",
	gt: ">",
	quot: '"',
	apos: "'",
	nbsp: " ",
};

function decode(s: string): string {
	return s.replace(/&(#x[\da-f]+|#\d+|[a-z]+);/gi, (m, e: string) => {
		if (e[0] !== "#") return NAMED[e.toLowerCase()] ?? m;
		const code =
			e[1] === "x" ? parseInt(e.slice(2), 16) : parseInt(e.slice(1), 10);
		return String.fromCodePoint(code);
	});
}

export function htmlToText(s: string): string {
	let t = decode(s);
	if (!/<[a-z][\s\S]*?>/i.test(t)) return t;
	t = t
		.replace(/<li\b[^>]*>/gi, "• ")
		.replace(/<\/(p|h[1-6]|ul|ol)>/gi, "\n\n")
		.replace(/<br\s*\/?>|<\/(div|li|tr)>/gi, "\n")
		.replace(/<[^>]*>/g, "");
	return decode(t)
		.replace(/[ \t]+\n/g, "\n")
		.replace(/\n{3,}/g, "\n\n")
		.trim();
}
