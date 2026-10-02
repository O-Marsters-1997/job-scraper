import { faker } from "@faker-js/faker";
import type { ResolvedURL, SourceInfo } from "@/types/source";
import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "@/types/sourceTarget";
import { ResolveError } from "../lib/resolveError";
import { updateCompanyTarget } from "./companies";
import { failIfRequested, hostMatches } from "./helpers";
import { seed } from "./seed";

let discoverySourceTargets: SourceTarget[] = seed.discoverySourceTargets;

const SOURCE_INFOS: SourceInfo[] = [
	{
		name: "greenhouse",
		label: "Greenhouse",
		kind: "board",
		role: "ats",
		url_prefix: "https://boards.greenhouse.io",
		filters: [],
	},
	{
		name: "lever",
		label: "Lever",
		kind: "board",
		role: "ats",
		url_prefix: "https://jobs.lever.co",
		filters: [],
	},
	{
		name: "ashby",
		label: "Ashby",
		kind: "board",
		role: "ats",
		url_prefix: "https://jobs.ashbyhq.com",
		filters: [],
	},
	{
		name: "workable",
		label: "Workable",
		kind: "board",
		role: "ats",
		url_prefix: "https://apply.workable.com",
		filters: [],
	},
	{
		name: "recruitee",
		label: "Recruitee",
		kind: "board",
		role: "ats",
		url_prefix: "https://recruitee.com",
		filters: [],
	},
	{
		name: "personio",
		label: "Personio",
		kind: "board",
		role: "ats",
		url_prefix: "https://personio.de",
		filters: [],
	},
	{
		name: "wis",
		label: "Work in Startups",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://workinstartups.com",
		filters: [
			{
				name: "loc",
				label: "Location",
				required: false,
				options: [
					{ value: "86383", label: "United Kingdom" },
					{ value: "86384", label: "London" },
				],
			},
		],
	},
	{
		name: "linkedin",
		label: "LinkedIn",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://www.linkedin.com/jobs",
		filters: [
			{ name: "location", label: "Location", required: false },
			{
				name: "recency",
				label: "Recency",
				required: false,
				options: [
					{ value: "day", label: "Past 24 hours" },
					{ value: "week", label: "Past week" },
					{ value: "month", label: "Past month" },
				],
			},
		],
	},
	{
		name: "indeed",
		label: "Indeed",
		kind: "url",
		role: "discovery",
		url_prefix: "https://www.indeed.com",
		filters: [],
	},
	{
		name: "remoteok",
		label: "RemoteOK",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://remoteok.com",
		filters: [],
	},
	{
		name: "remotive",
		label: "Remotive",
		kind: "filter",
		role: "discovery",
		url_prefix: "https://remotive.com",
		filters: [],
	},
];

export function getSources(): SourceInfo[] {
	return SOURCE_INFOS;
}

export function resolveUrl(url: string): ResolvedURL {
	let parsed: URL;
	try {
		parsed = new URL(url);
	} catch {
		throw new ResolveError(
			"could not recognise a supported board or search page in that URL",
		);
	}
	const source = SOURCE_INFOS.find((s) =>
		hostMatches(parsed.hostname, new URL(s.url_prefix).hostname),
	);
	if (source?.role === "ats") {
		const value = parsed.pathname.split("/").filter(Boolean).pop() ?? "";
		return {
			kind: "ats",
			source: source.name,
			value,
			filters: {},
			dropped: [],
			url,
		};
	}
	if (source?.kind === "filter" && source.filters.length > 0) {
		const filters = Object.fromEntries(
			source.filters
				.map((f): [string, string] => [
					f.name,
					parsed.searchParams.get(f.name) ?? "",
				])
				.filter(([, v]) => v),
		);
		return {
			kind: "search",
			source: source.name,
			value: parsed.searchParams.get("keywords") ?? "",
			filters,
			dropped: ["trk"],
			url,
		};
	}
	if (source?.kind === "url") {
		return {
			kind: "search",
			source: source.name,
			value: url,
			filters: {},
			dropped: [],
			url,
		};
	}
	if (source) {
		throw new ResolveError(
			`${source.label} searches aren't supported yet — use Build from fields`,
		);
	}
	throw new ResolveError(
		"could not recognise a supported board or search page in that URL",
	);
}

export function getSourceTargets(): SourceTarget[] {
	return discoverySourceTargets;
}

export function createSourceTarget(
	payload: CreateSourceTargetPayload,
): SourceTarget | null {
	failIfRequested("createSourceTarget");
	const exists = getSourceTargets().some(
		(t) => t.Source === payload.source && t.Value === payload.value,
	);
	if (exists) return null;
	const target: SourceTarget = {
		ID: faker.string.uuid(),
		UserID: "user-1",
		Source: payload.source,
		Value: payload.value,
		Enabled: payload.enabled ?? true,
		Filters: payload.filters ?? {},
		RunStatus: "succeeded",
		LastRunAt: new Date().toISOString(),
		LastRunError: "",
		DisabledReason: "",
		URL: SOURCE_INFOS.find((s) => s.name === payload.source)?.url_prefix ?? "",
	};
	discoverySourceTargets = [...discoverySourceTargets, target];
	return target;
}

export function updateSourceTarget(
	id: string,
	patch: UpdateSourceTargetPayload,
): SourceTarget {
	const companyTarget = updateCompanyTarget(id, patch);
	if (companyTarget) return companyTarget;
	const idx = discoverySourceTargets.findIndex((t) => t.ID === id);
	if (idx === -1) throw new Error("Source target not found");
	const updated: SourceTarget = {
		...discoverySourceTargets[idx]!,
		Enabled: patch.enabled ?? discoverySourceTargets[idx]!.Enabled,
	};
	discoverySourceTargets = [
		...discoverySourceTargets.slice(0, idx),
		updated,
		...discoverySourceTargets.slice(idx + 1),
	];
	return updated;
}

export function deleteSourceTarget(id: string): void {
	discoverySourceTargets = discoverySourceTargets.filter((t) => t.ID !== id);
}

export function rerunSourceTarget(id: string): SourceTarget {
	const idx = discoverySourceTargets.findIndex((t) => t.ID === id);
	if (idx === -1) {
		const existing = getSourceTargets().find((t) => t.ID === id);
		if (!existing) throw new Error("Source target not found");
		return {
			...existing,
			RunStatus: "succeeded",
			LastRunAt: new Date().toISOString(),
		};
	}
	const updated: SourceTarget = {
		...discoverySourceTargets[idx]!,
		RunStatus: "succeeded",
		LastRunAt: new Date().toISOString(),
	};
	discoverySourceTargets = [
		...discoverySourceTargets.slice(0, idx),
		updated,
		...discoverySourceTargets.slice(idx + 1),
	];
	return updated;
}
