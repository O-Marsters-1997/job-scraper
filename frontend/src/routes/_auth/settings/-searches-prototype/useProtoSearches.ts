import { createSignal } from "solid-js";
import { resolveBoard } from "@/api/sources";
import { useCompanies } from "@/hooks/useCompanies";
import { useSources } from "@/hooks/useSources";
import { useSourceTargets } from "@/hooks/useSourceTargets";
import type { SourceTarget } from "@/types/sourceTarget";

export type SourceTab = "ats" | "boards";

export interface CompanyBoardRow {
	target: SourceTarget;
	companyId: string | null;
	name: string;
	boardUrl: string;
	open: number;
	relevant: number;
}

const BOARD_URL: Record<string, (token: string) => string> = {
	greenhouse: (t) => `https://boards.greenhouse.io/${t}`,
	lever: (t) => `https://jobs.lever.co/${t}`,
	ashby: (t) => `https://jobs.ashbyhq.com/${t}`,
	workable: (t) => `https://apply.workable.com/${t}`,
	recruitee: (t) => `https://${t}.recruitee.com`,
	personio: (t) => `https://${t}.jobs.personio.de`,
};

export const boardUrlFor = (source: string, token: string) =>
	BOARD_URL[source]?.(token) ?? "";

const hash = (s: string) =>
	[...s].reduce((h, c) => (Math.imul(h, 31) + c.charCodeAt(0)) >>> 0, 7);

const NAME_A = [
	"Arc",
	"Blue",
	"Bright",
	"Cedar",
	"Copper",
	"Delta",
	"Echo",
	"Fern",
	"Flux",
	"Granite",
	"Harbor",
	"Iris",
	"Juniper",
	"Kite",
	"Lumen",
	"Maple",
	"Nimbus",
	"Onyx",
	"Pine",
	"Quartz",
	"River",
	"Slate",
	"Tidal",
	"Umber",
	"Vector",
	"Willow",
	"Zephyr",
];
const NAME_B = [
	"Labs",
	"Health",
	"Pay",
	"AI",
	"Works",
	"Bank",
	"Robotics",
	"Energy",
	"Data",
	"Cloud",
	"Logistics",
	"Bio",
];
const ATS = Object.keys(BOARD_URL);

const SYNTHETIC_NAMES = new Map<string, string>();

function syntheticTargets(): SourceTarget[] {
	const out: SourceTarget[] = [];
	for (const a of NAME_A) {
		for (const b of NAME_B) {
			const name = `${a} ${b}`;
			const h = hash(name);
			if (h % 3 === 0) continue;
			SYNTHETIC_NAMES.set(`proto-${h}`, name);
			out.push({
				ID: `proto-${h}`,
				UserID: "user-1",
				Source: ATS[h % ATS.length] ?? "greenhouse",
				Value: `${a}${b}`.toLowerCase(),
				Enabled: h % 7 !== 0,
				Filters: {},
				RunStatus: "succeeded",
				LastRunAt: new Date(Date.now() - (h % 4000) * 60000).toISOString(),
				LastRunError: "",
			});
		}
	}
	return out;
}

const SYNTHETIC = syntheticTargets();

const titleCase = (token: string) =>
	token.replace(
		/(^|[-_])(\w)/g,
		(_, sep, c) => `${sep ? " " : ""}${c.toUpperCase()}`,
	);

export function useProtoSearches() {
	const query = useSourceTargets();
	const sources = useSources();
	const companies = useCompanies();
	const [added, setAdded] = createSignal<SourceTarget[]>([]);
	const [patches, setPatches] = createSignal<
		Record<string, Partial<SourceTarget>>
	>({});
	const [removed, setRemoved] = createSignal<string[]>([]);

	const info = (name: string) =>
		(sources.data ?? []).find((s) => s.name === name);
	const label = (name: string) => info(name)?.label ?? name;

	const all = () =>
		[...added(), ...(query.data ?? []), ...SYNTHETIC]
			.filter((t) => !removed().includes(t.ID))
			.map((t) => ({ ...t, ...patches()[t.ID] }));

	const tabOf = (t: SourceTarget): SourceTab =>
		info(t.Source)?.role === "ats" ? "ats" : "boards";
	const list = (tab: SourceTab) => all().filter((t) => tabOf(t) === tab);

	const companyRows = (): CompanyBoardRow[] => {
		const known = companies.data ?? [];
		return list("ats").map((t) => {
			const company = known.find(
				(c) =>
					c.TargetID === t.ID ||
					(c.ATSSource === t.Source && c.ATSToken === t.Value),
			);
			const h = hash(t.Value);
			const open = company ? Math.max(company.JobCount, h % 60) : h % 180;
			return {
				target: t,
				companyId: company?.ID ?? known[h % known.length]?.ID ?? null,
				name: company?.Name ?? SYNTHETIC_NAMES.get(t.ID) ?? titleCase(t.Value),
				boardUrl: boardUrlFor(t.Source, t.Value),
				open,
				relevant: Math.round((open * ((h >> 3) % 45)) / 100),
			};
		});
	};

	const patch = (id: string, p: Partial<SourceTarget>) =>
		setPatches((prev) => ({ ...prev, [id]: { ...prev[id], ...p } }));

	const run = (id: string) => {
		patch(id, { RunStatus: "queued" });
		setTimeout(() => patch(id, { RunStatus: "running" }), 900);
		setTimeout(
			() =>
				patch(id, {
					RunStatus: "succeeded",
					LastRunAt: new Date().toISOString(),
				}),
			2600,
		);
	};

	const add = (source: string, value: string, filters = {}) => {
		const t: SourceTarget = {
			ID: crypto.randomUUID(),
			UserID: "user-1",
			Source: source,
			Value: value,
			Enabled: true,
			Filters: filters,
			RunStatus: "idle",
			LastRunAt: null,
			LastRunError: "",
		};
		setAdded((prev) => [t, ...prev]);
		if (info(source)?.role !== "ats") run(t.ID);
		return t;
	};

	return {
		query,
		sources,
		info,
		label,
		list,
		companyRows,
		tabOf,
		exists: (source: string, value: string) =>
			all().some((t) => t.Source === source && t.Value === value),
		add,
		run,
		toggle: (t: SourceTarget) => patch(t.ID, { Enabled: !t.Enabled }),
		remove: (id: string) => setRemoved((prev) => [...prev, id]),
		resolveBoard,
	};
}

export type ProtoSearches = ReturnType<typeof useProtoSearches>;

export const relativeTime = (iso: string | null) => {
	if (!iso) return "Never";
	const mins = Math.round((Date.now() - new Date(iso).getTime()) / 60000);
	if (mins < 1) return "Just now";
	if (mins < 60) return `${mins}m ago`;
	const hours = Math.round(mins / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.round(hours / 24)}d ago`;
};
