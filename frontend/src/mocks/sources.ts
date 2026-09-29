import { faker } from "@faker-js/faker";
import type { ResolvedBoard, SourceInfo } from "@/types/source";
import type {
	CreateSourceTargetPayload,
	SourceTarget,
	UpdateSourceTargetPayload,
} from "@/types/sourceTarget";
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
				name: "region",
				label: "Region",
				required: false,
				options: [
					{ value: "uk", label: "UK" },
					{ value: "us", label: "US" },
					{ value: "remote", label: "Remote" },
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

export function resolveBoard(url: string): ResolvedBoard | null {
	let hostname: string;
	try {
		hostname = new URL(url).hostname;
	} catch {
		return null;
	}
	const match = SOURCE_INFOS.find(
		(s) =>
			s.kind === "board" &&
			hostMatches(hostname, new URL(s.url_prefix).hostname),
	);
	if (!match) return null;
	const value = url.replace(/\/$/, "").split("/").pop() ?? hostname;
	return { source: match.name, value };
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
