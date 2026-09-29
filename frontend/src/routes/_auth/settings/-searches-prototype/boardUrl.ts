export interface ParamSpec {
	key: string;
	label: string;
	options?: Record<string, string>;
	placeholder?: string;
}

export interface BoardSpec {
	source: string;
	label: string;
	hosts: string[];
	canonicalUrl: string;
	pathPattern: RegExp;
	keywordsKey: string;
	params: ParamSpec[];
	drop: string[];
	pageKey: string;
	pageStep: number;
	pageFirst: number;
	fixed?: Record<string, string>;
}

export const BOARD_SPECS: BoardSpec[] = [
	{
		source: "linkedin",
		label: "LinkedIn",
		hosts: ["linkedin.com"],
		canonicalUrl: "https://www.linkedin.com/jobs/search/",
		pathPattern: /^\/jobs\/(search|search-results)\/?$/,
		keywordsKey: "keywords",
		params: [
			{ key: "location", label: "Location", placeholder: "e.g. London" },
			{
				key: "f_TPR",
				label: "Posted",
				options: {
					r86400: "Past 24 hours",
					r604800: "Past week",
					r2592000: "Past month",
				},
			},
			{
				key: "f_WT",
				label: "Workplace",
				options: { "1": "On-site", "2": "Remote", "3": "Hybrid" },
			},
			{
				key: "f_E",
				label: "Experience",
				options: {
					"1": "Internship",
					"2": "Entry level",
					"3": "Associate",
					"4": "Mid-Senior",
					"5": "Director",
					"6": "Executive",
				},
			},
			{
				key: "f_JT",
				label: "Job type",
				options: {
					F: "Full-time",
					P: "Part-time",
					C: "Contract",
					T: "Temporary",
					I: "Internship",
				},
			},
			{ key: "geoId", label: "Geo ID" },
			{ key: "f_D", label: "Distance (mi)" },
			{ key: "f_C", label: "Company ID" },
		],
		drop: [
			"currentJobId",
			"origin",
			"referralSearchId",
			"refresh",
			"position",
			"pageNum",
			"trk",
			"trackingId",
		],
		pageKey: "start",
		pageStep: 25,
		pageFirst: 0,
	},
	{
		source: "indeed",
		label: "Indeed",
		hosts: ["indeed.com", "indeed.co.uk"],
		canonicalUrl: "https://www.indeed.com/jobs",
		pathPattern: /^\/jobs\/?$/,
		keywordsKey: "q",
		params: [
			{ key: "l", label: "Location", placeholder: "e.g. London" },
			{
				key: "fromage",
				label: "Posted",
				options: {
					"1": "Past 24 hours",
					"3": "Past 3 days",
					"7": "Past week",
					"14": "Past 2 weeks",
				},
			},
			{ key: "radius", label: "Radius (mi)" },
		],
		drop: ["vjk", "from", "advn", "vjs", "sc"],
		pageKey: "start",
		pageStep: 10,
		pageFirst: 0,
	},
	{
		source: "wis",
		label: "Work in Startups",
		hosts: ["workinstartups.com"],
		canonicalUrl: "https://workinstartups.com/search",
		pathPattern: /^\/search\/?$/,
		keywordsKey: "q",
		params: [{ key: "w", label: "Region", placeholder: "e.g. uk" }],
		drop: [],
		pageKey: "p",
		pageStep: 1,
		pageFirst: 1,
		fixed: { per_page: "50" },
	},
];

export interface ParsedParam {
	key: string;
	value: string;
	label: string;
	display: string;
}

export type ParseOk = {
	ok: true;
	spec: BoardSpec;
	origin: string;
	keywords: string;
	params: ParsedParam[];
	kept: ParsedParam[];
	dropped: { key: string; value: string }[];
	pastedPage: number | null;
};

export type ParseResult = { ok: false; reason: string } | ParseOk;

const hostMatches = (host: string, want: string) =>
	host === want || host.endsWith(`.${want}`);

export const specFor = (source: string) =>
	BOARD_SPECS.find((s) => s.source === source);

const displayFor = (p: ParamSpec | undefined, value: string) =>
	p?.options
		? value
				.split(",")
				.map((v) => p.options?.[v] ?? v)
				.join(", ")
		: value;

const toParam = (spec: BoardSpec | undefined, key: string, value: string) => {
	const p = spec?.params.find((x) => x.key === key);
	return { key, value, label: p?.label ?? key, display: displayFor(p, value) };
};

export function parseBoardUrl(raw: string): ParseResult {
	let url: URL;
	try {
		url = new URL(raw.trim());
	} catch {
		return { ok: false, reason: "That isn't a full URL. Include https://" };
	}
	const spec = BOARD_SPECS.find((s) =>
		s.hosts.some((h) => hostMatches(url.hostname, h)),
	);
	if (!spec) {
		return {
			ok: false,
			reason: `${url.hostname} isn't a supported job board. LinkedIn, Indeed and Work in Startups are.`,
		};
	}
	if (!spec.pathPattern.test(url.pathname)) {
		return {
			ok: false,
			reason: `That's a ${spec.label} page, but not a search results page.`,
		};
	}

	const params: ParsedParam[] = [];
	const kept: ParsedParam[] = [];
	const dropped: { key: string; value: string }[] = [];
	let pastedPage: number | null = null;

	for (const [key, value] of url.searchParams) {
		if (key === spec.keywordsKey || spec.fixed?.[key] !== undefined) continue;
		if (key === spec.pageKey) {
			const n = Number(value);
			if (Number.isFinite(n)) {
				pastedPage = Math.floor((n - spec.pageFirst) / spec.pageStep) + 1;
			}
			continue;
		}
		if (spec.drop.includes(key) || key.startsWith("utm_")) {
			dropped.push({ key, value });
			continue;
		}
		const known = spec.params.some((x) => x.key === key);
		(known ? params : kept).push(toParam(spec, key, value));
	}

	return {
		ok: true,
		spec,
		origin: url.origin,
		keywords: url.searchParams.get(spec.keywordsKey) ?? "",
		params,
		kept,
		dropped,
		pastedPage: pastedPage && pastedPage > 1 ? pastedPage : null,
	};
}

export interface SearchDraft {
	source: string;
	keywords: string;
	filters: Record<string, string>;
	origin?: string;
}

export function draftFromParse(r: ParseOk): SearchDraft {
	const filters: Record<string, string> = {};
	for (const p of [...r.params, ...r.kept]) filters[p.key] = p.value;
	return {
		source: r.spec.source,
		keywords: r.keywords,
		filters,
		origin: r.origin,
	};
}

export function buildBoardUrl(d: SearchDraft, page = 1): string {
	const spec = specFor(d.source);
	if (!spec) return "";
	const canonical = new URL(spec.canonicalUrl);
	const url = new URL(canonical.pathname, d.origin ?? canonical.origin);
	if (d.keywords) url.searchParams.set(spec.keywordsKey, d.keywords);
	for (const [k, v] of Object.entries(d.filters)) {
		if (v) url.searchParams.set(k, v);
	}
	for (const [k, v] of Object.entries(spec.fixed ?? {})) {
		url.searchParams.set(k, v);
	}
	if (page > 1) {
		url.searchParams.set(
			spec.pageKey,
			String(spec.pageFirst + (page - 1) * spec.pageStep),
		);
	}
	return url.toString();
}

export function describeFilters(d: SearchDraft): ParsedParam[] {
	const spec = specFor(d.source);
	return Object.entries(d.filters)
		.filter(([, v]) => v)
		.map(([key, value]) => toParam(spec, key, value));
}

export function pagingNote(source: string) {
	const spec = specFor(source);
	if (!spec) return "";
	const next =
		spec.pageStep === 1 ? "2, 3" : `${spec.pageStep}, ${spec.pageStep * 2}`;
	return `then ${spec.pageKey}=${next}, …`;
}

export function draftFromTarget(t: {
	Source: string;
	Value: string;
	Filters: Record<string, string>;
}): SearchDraft {
	if (t.Value.startsWith("http")) {
		const r = parseBoardUrl(t.Value);
		if (r.ok) return draftFromParse(r);
	}
	const spec = specFor(t.Source);
	const filters: Record<string, string> = {};
	for (const [k, v] of Object.entries(t.Filters)) {
		const mapped =
			spec?.params.find((p) => p.label.toLowerCase() === k.toLowerCase())
				?.key ?? k;
		filters[mapped] = v;
	}
	return { source: t.Source, keywords: t.Value, filters };
}
