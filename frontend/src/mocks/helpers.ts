export function slugify(name: string): string {
	return name
		.toLowerCase()
		.replace(/\s+/g, "-")
		.replace(/[^a-z0-9-]/g, "");
}

export function hostMatches(hostname: string, host: string): boolean {
	return hostname === host || hostname.endsWith(`.${host}`);
}

export function humanizeSlug(slug: string): string {
	return slug
		.split(/[-_]/)
		.filter(Boolean)
		.map((w) => w[0]!.toUpperCase() + w.slice(1))
		.join(" ");
}

export function failIfRequested(op: string): void {
	const requested = globalThis.localStorage?.getItem("mock-fail") ?? "";
	if (requested.split(",").includes(op)) throw new Error(`mock ${op} failed`);
}
