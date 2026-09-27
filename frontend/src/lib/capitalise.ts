export const capitalise = (value: string) =>
	value
		.trim()
		.replace(/(^|\s)(\p{Ll})/gu, (_, gap, c) => gap + c.toUpperCase());

export const uniqueCapitalised = (values: string[]) => {
	const seen = new Set<string>();
	return values.map(capitalise).filter((v) => {
		const key = v.toLowerCase();
		if (!v || seen.has(key)) return false;
		seen.add(key);
		return true;
	});
};
