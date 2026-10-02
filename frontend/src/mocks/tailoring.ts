import type {
	CVHeading,
	Draft,
	DraftInput,
	DraftLayout,
	DraftProvenance,
	DraftRef,
	Explanation,
	HeadingMapping,
	LayoutBlock,
	LayoutRun,
	SlotEdit,
	SuggestDone,
	Suggestion,
	SuggestRequest,
} from "@/types/tailoring";
import { KeptDraftExistsError } from "../lib/tailoring";
import { getExperience } from "./experience";
import { getJobs } from "./jobs";

const savedHeadings = new Map<string, HeadingMapping[]>();

export function getHeadings(docId: string, tabId: string): CVHeading[] {
	const saved = savedHeadings.get(`${docId}/${tabId}`);
	const roles: { text: string; match: string | null; slots: number }[] = [
		{
			text: "Senior Backend Engineer, Acme Ltd",
			match: "position-1",
			slots: 2,
		},
		{ text: "Software Engineer, Globex", match: "position-2", slots: 1 },
	];
	return roles.map((r) => {
		const s = saved?.find((m) => m.headingText === r.text);
		return {
			text: r.text,
			positionId: s ? s.positionId : r.match,
			confirmed: s !== undefined,
			slotCount: r.slots,
		};
	});
}

export function saveHeadings(
	docId: string,
	tabId: string,
	mappings: HeadingMapping[],
): HeadingMapping[] {
	savedHeadings.set(`${docId}/${tabId}`, mappings);
	return mappings;
}

export function getSuggestions(): Suggestion[] {
	const scores = [0.81, 0.64, 0.42, 0.17];
	return getExperience().flatMap((p) => {
		const slots = p.id === "position-1" ? 2 : 3;
		return p.achievements.map((a, i) => ({
			achievementId: a.id,
			positionId: p.id,
			text: a.text,
			score: scores[i] ?? 0.1,
			preselected: i < slots,
			state: i === 2 ? "unclear" : i > 2 ? "low" : "fit",
		}));
	});
}

const mockDrafts = new Map<string, { draft: Draft; polls: number }>();

const MOCK_DOC_URL = "https://docs.google.com/document/d/mock-draft/edit";

const MOCK_PROFILE =
	"Backend engineer with six years of experience in platform work, from first prototype to on-call rota.";

const MOCK_ACHIEVEMENTS = [
	{
		id: "ach-1",
		positionId: "position-1",
		text: "Cut p99 latency of the billing API from 900ms to 240ms.",
	},
	{
		id: "ach-2",
		positionId: "position-1",
		text: "Led the migration of twelve services to Kubernetes, reducing deploy time and the pager load for a team of eight across two time zones.",
	},
	{
		id: "ach-3",
		positionId: "position-2",
		text: "Built the internal ledger service in Go and Postgres.",
	},
];

const MOCK_BULLET_TEXT: Record<string, string> = {
	s1: MOCK_ACHIEVEMENTS[0]?.text ?? "",
	s2: MOCK_ACHIEVEMENTS[1]?.text ?? "",
	s3: MOCK_ACHIEVEMENTS[2]?.text ?? "",
};

function mockProvenance(): DraftProvenance {
	const bullet = (slotId: string, achievement: number) => ({
		slotId,
		achievements: MOCK_ACHIEVEMENTS.slice(achievement, achievement + 1),
	});
	return {
		positions: [
			{
				positionId: "position-1",
				employer: "Acme Ltd",
				title: "Senior Backend Engineer",
				bullets: [bullet("s1", 0), bullet("s2", 1)],
			},
			{
				positionId: "position-2",
				employer: "Globex",
				title: "Software Engineer",
				bullets: [bullet("s3", 2)],
			},
		],
		profile: { slotId: "profile" },
	};
}

function mockContent(): Pick<Draft, "content" | "base"> {
	const p = getExperience()[0];
	const texts = p?.achievements.slice(0, 3).map((a) => a.text) ?? [];
	const bullet = (text: string) => ({ text, achievementIds: [] });
	const content = (bullets: string[], skills: string[], profile: string) => ({
		profile,
		skills,
		positions: p ? [{ positionId: p.id, bullets: bullets.map(bullet) }] : [],
	});
	const [first = "", second = "", third = ""] = texts;
	return {
		base: content(
			[first, second, third],
			["Go", "SQL"],
			"Backend engineer with six years of experience.",
		),
		content: content(
			[third, `${first} using Kubernetes`, "Led the on-call rota"],
			["Go", "Kubernetes"],
			"Backend engineer with six years of experience in platform work.",
		),
	};
}

export function createDraft(input: DraftInput): DraftRef {
	const id = `draft-${mockDrafts.size + 1}`;
	mockDrafts.set(id, {
		draft: {
			id,
			jobId: input.jobId,
			status: "pending",
			outcome: null,
			keptAs: "",
			draftDocUrl: null,
			lastError: "",
			createdAt: new Date().toISOString(),
			findings: [],
			provenance: null,
			content: null,
			base: null,
		},
		polls: 0,
	});
	return { id };
}

function mockEntry(id: string) {
	const entry = mockDrafts.get(id);
	if (!entry) throw new Error(`no mock draft ${id}`);
	return entry;
}

export function getDraft(id: string): Draft {
	const entry = mockEntry(id);
	entry.polls += 1;
	if (entry.polls >= 2 && entry.draft.status === "keeping") {
		entry.draft = {
			...entry.draft,
			status: "ready",
			outcome: "kept",
			keptAs: "doc",
		};
	}
	if (entry.polls >= 3 && ["pending", "running"].includes(entry.draft.status)) {
		entry.draft = {
			...entry.draft,
			status: "ready",
			draftDocUrl: MOCK_DOC_URL,
			provenance: mockProvenance(),
			...mockContent(),
			findings: [
				{
					check: "grounding",
					severity: "block",
					message: 'number "40%" does not appear in the cited Achievements',
				},
				{
					check: "skills",
					severity: "info",
					message: 'job skill "Terraform" has no source in your CV or Bank',
				},
			],
		};
	} else if (entry.polls === 2) {
		entry.draft = { ...entry.draft, status: "running" };
	}
	return entry.draft;
}

export function getJobDrafts(jobId: string): Draft[] {
	return [...mockDrafts.values()]
		.map((e) => e.draft)
		.filter((d) => d.jobId === jobId)
		.reverse();
}

export function keepDraft(id: string): Draft {
	const entry = mockEntry(id);
	const other = getJobDrafts(entry.draft.jobId).find(
		(d) => d.outcome === "kept" && d.id !== id,
	);
	if (other) throw new KeptDraftExistsError();
	entry.polls = 0;
	entry.draft = { ...entry.draft, status: "keeping" };
	return entry.draft;
}

export function discardDraft(id: string): Draft {
	const entry = mockEntry(id);
	entry.draft = { ...entry.draft, outcome: "discarded", draftDocUrl: null };
	return entry.draft;
}

const MOCK_TWO_PAGES_CHARS = 1500;

export function saveDraftSlots(id: string, slots: SlotEdit[]): Draft {
	const entry = mockEntry(id);
	if (!entry.draft.provenance) throw new Error(`mock draft ${id} is not ready`);
	const chars = slots.reduce((n, s) => n + s.text.length, 0);
	const findings = entry.draft.findings.filter((f) => f.check !== "page_count");
	if (chars > MOCK_TWO_PAGES_CHARS)
		findings.push({
			check: "page_count",
			severity: "block",
			message: "the Draft is 2 pages; the base CV is 1",
		});
	entry.draft = {
		...entry.draft,
		findings,
		provenance: mockProvenance(),
	};
	return entry.draft;
}

function seedDraft(id: string, outcome: Draft["outcome"]) {
	mockDrafts.set(id, {
		polls: 0,
		draft: {
			id,
			jobId: getJobs().at(-1)?.ID ?? "",
			status: "ready",
			outcome,
			keptAs: outcome === "kept" ? "doc" : "",
			draftDocUrl: MOCK_DOC_URL,
			lastError: "",
			createdAt: "2026-10-01T09:00:00Z",
			findings: [
				{
					check: "grounding",
					severity: "warn",
					slotId: "s2",
					message: "the number of twelve services is not in the Achievement",
				},
				{
					check: "slot_length",
					severity: "info",
					message: "Skills run to two lines on this Doc",
				},
				{
					check: "skills",
					severity: "info",
					message: 'job skill "Terraform" has no source in your CV or Bank',
				},
			],
			provenance: mockProvenance(),
			...mockContent(),
		},
	});
}

seedDraft("draft-ready", null);
seedDraft("draft-kept", "kept");

const mockRun = (text: string, over: Partial<LayoutRun> = {}): LayoutRun => ({
	text,
	font: "Calibri",
	size: 11,
	bold: false,
	italic: false,
	underline: false,
	color: "",
	link: "",
	...over,
});

const mockBlock = (over: Partial<LayoutBlock>): LayoutBlock => ({
	slotId: "",
	section: "",
	align: "left",
	lineSpacing: 115,
	spaceAbove: 0,
	spaceBelow: 0,
	indentStart: 0,
	indentFirstLine: 0,
	borderTop: null,
	borderBottom: null,
	tabStops: [],
	bullet: null,
	runs: [],
	...over,
});

const MOCK_RULE = { width: 0.75, color: "#000000", padding: 1, dash: "solid" };

const mockSection = (title: string): LayoutBlock =>
	mockBlock({
		spaceAbove: 8,
		borderBottom: MOCK_RULE,
		runs: [mockRun(title, { bold: true, size: 12 })],
	});

const mockEmployer = (left: string, right: string): LayoutBlock =>
	mockBlock({
		spaceAbove: 4,
		tabStops: [{ offset: 487, alignment: "end" }],
		runs: [mockRun(`${left}\t`, { bold: true }), mockRun(right)],
	});

const mockBullet = (slotId: string): LayoutBlock =>
	mockBlock({
		slotId,
		indentStart: 36,
		indentFirstLine: 18,
		bullet: { glyph: "\u2022", level: 0, size: 11 },
		runs: [mockRun(MOCK_BULLET_TEXT[slotId] ?? "")],
	});

const mockSkillRow = (label: string, value: string): LayoutBlock =>
	mockBlock({
		section: "skills",
		tabStops: [{ offset: 90, alignment: "start" }],
		runs: [mockRun(`${label}\t`, { bold: true }), mockRun(value)],
	});

export function getDraftLayout(): DraftLayout {
	return {
		page: {
			width: 595.28,
			height: 841.89,
			marginTop: 36,
			marginBottom: 36,
			marginLeft: 54,
			marginRight: 54,
		},
		blocks: [
			mockBlock({
				align: "center",
				runs: [mockRun("Jordan Example", { bold: true, size: 20 })],
			}),
			mockBlock({
				align: "center",
				runs: [
					mockRun("London, UK  |  jordan@example.com  |  example.com/jordan"),
				],
			}),
			mockBlock({ runs: [mockRun("\n", { size: 6 })] }),
			mockSection("Profile"),
			mockBlock({
				slotId: "profile",
				align: "justify",
				spaceAbove: 3,
				runs: [mockRun(MOCK_PROFILE)],
			}),
			mockSection("Experience"),
			mockEmployer("Senior Backend Engineer, Acme Ltd", "2021 - Present"),
			mockBullet("s1"),
			mockBullet("s2"),
			mockEmployer("Software Engineer, Globex", "2018 - 2021"),
			mockBullet("s3"),
			mockSection("Skills"),
			mockSkillRow("Languages", "Go, SQL, TypeScript"),
			mockSkillRow("Tools", "Kubernetes, Postgres, Terraform"),
		],
	};
}

const MOCK_WORD_MS = 25;

function mockSuggestion(req: SuggestRequest): string {
	const words = req.text.split(/\s+/).filter(Boolean);
	if (req.action === "fit") {
		const kept: string[] = [];
		for (const w of words) {
			if ([...kept, w].join(" ").length > req.maxChars) break;
			kept.push(w);
		}
		return kept.join(" ");
	}
	if (req.action === "verb") return ["Led", ...words.slice(1)].join(" ");
	return words.slice(0, Math.ceil(words.length * 0.75)).join(" ");
}

export async function streamSuggestion(
	req: SuggestRequest,
	onDelta: (text: string) => void,
	signal: AbortSignal,
): Promise<SuggestDone> {
	const text = mockSuggestion(req);
	const words = text.split(" ");
	for (const [i, w] of words.entries()) {
		signal.throwIfAborted();
		onDelta(i === 0 ? w : ` ${w}`);
		await new Promise((resolve) => setTimeout(resolve, MOCK_WORD_MS));
	}
	signal.throwIfAborted();
	return { text, findings: [] };
}

export async function explainAchievement(
	_achievementId: string,
): Promise<Explanation> {
	await new Promise((resolve) => setTimeout(resolve, MOCK_WORD_MS * 10));
	return {
		text: "The job asks for on-call ownership; this bullet doesn't show it.",
	};
}
