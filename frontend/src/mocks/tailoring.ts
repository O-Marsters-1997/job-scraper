import type {
	BulletLean,
	CVHeading,
	Draft,
	DraftContent,
	DraftInput,
	DraftInsights,
	DraftLayout,
	DraftPositionEdit,
	DraftProvenance,
	DraftRef,
	DraftRoles,
	ExperienceMatch,
	HeadingMapping,
	LayoutBlock,
	LayoutRun,
	ProfileRedraftInput,
	SkillGroup,
	SlotEdit,
	SuggestDone,
	Suggestion,
	SuggestRequest,
} from "@/types/tailoring";
import { KeptDraftExistsError } from "../lib/tailoring";
import { getExperience } from "./experience";
import { getJobs } from "./jobs";
import { seed } from "./seed";

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

export function getDraftLean(): BulletLean[] {
	return [];
}

export function getDraftRoles(): DraftRoles {
	const role = (
		positionId: string,
		title: string,
		employer: string,
		budget: number,
		used: number,
	) => ({
		positionId,
		employer,
		title,
		budget,
		bullets: [
			{
				slotId: `${positionId}-s1`,
				achievementId: null,
				text: title,
				lines: used,
				lean: null,
				state: null,
				onSource: true,
				replacedFrom: null,
			},
		],
		bank: [],
	});
	return {
		draftId: "draft-ready",
		kept: false,
		spills: false,
		roles: [
			role("position-1", "Senior Backend Engineer", "Acme Ltd", 4, 6),
			role("position-2", "Software Engineer", "Globex", 3, 2),
		],
	};
}

export function getDraftInsights(): DraftInsights {
	return { lines: [], stale: false };
}

export function getExperienceMatch(): ExperienceMatch {
	return { score: 0.34 };
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
			lines: 1 + (i % 2),
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
		id: "achievement-1",
		positionId: "position-1",
		text: "Cut p99 latency of the billing API from 900ms to 240ms.",
	},
	{
		id: "achievement-2",
		positionId: "position-1",
		text: "Led the migration of twelve services to Kubernetes, reducing deploy time and the pager load for a team of eight across two time zones.",
	},
	{
		id: "achievement-3",
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
	const content = (
		bullets: string[],
		skillGroups: { label: string; items: string[] }[],
		profile: string,
	) => ({
		profile,
		skills: skillGroups.flatMap((g) => g.items),
		skillGroups,
		positions: p
			? [
					{
						positionId: p.id,
						headingText: "",
						linked: true,
						bullets: bullets.map(bullet),
					},
				]
			: [],
	});
	const [first = "", second = "", third = ""] = texts;
	const generated = (positionId: string, slotIds: string[]) => ({
		positionId,
		headingText: "",
		linked: true,
		bullets: slotIds.map((slotId) => ({
			text: MOCK_BULLET_TEXT[slotId] ?? "",
			achievementIds: MOCK_ACHIEVEMENTS.filter(
				(a) => a.text === MOCK_BULLET_TEXT[slotId],
			).map((a) => a.id),
		})),
	});
	return {
		base: content(
			[first, second, third],
			[
				{ label: "Languages", items: ["Go", "SQL"] },
				{ label: "Tools", items: ["Postgres"] },
			],
			"Backend engineer with six years of experience.",
		),
		content: {
			...content(
				[],
				[
					{ label: "Languages", items: ["Go", "SQL"] },
					{ label: "Tools", items: ["Postgres", "Kubernetes"] },
				],
				"Backend engineer with six years of experience in platform work.",
			),
			positions: [
				generated("position-1", ["s1", "s2"]),
				generated("position-2", ["s3"]),
				{
					positionId: "",
					headingText: "Contractor, Initech",
					linked: false,
					bullets: [],
				},
			],
		},
	};
}

export function createDraft(input: DraftInput): DraftRef {
	const id = `draft-${mockDrafts.size + 1}`;
	mockDrafts.set(id, {
		draft: {
			id,
			jobId: input.jobId,
			mode: input.mode,
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
			skillsEditable: false,
			insights: null,
			baseDocId: input.docId,
			baseTabId: input.tabId,
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
			skillsEditable: true,
			insights: null,
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

export const MOCK_REDRAFT_REJECTED =
	"the redraft was rejected by the guardrails";

export function redraftProfile(id: string, input: ProfileRedraftInput): Draft {
	const entry = mockEntry(id);
	const { content } = entry.draft;
	if (!content) throw new Error(`mock draft ${id} is not ready`);
	if (input.steer?.includes("reject")) throw new Error(MOCK_REDRAFT_REJECTED);
	const steer = input.steer ? ` (${input.steer})` : "";
	entry.draft = {
		...entry.draft,
		content: { ...content, profile: `Redrafted profile${steer}.` },
	};
	return entry.draft;
}

const MOCK_TWO_PAGES_CHARS = 1500;

export function saveDraftSlots(
	id: string,
	slots: SlotEdit[],
	skills?: SkillGroup[],
	positions: DraftPositionEdit[] = [],
	roles?: string[],
): Draft {
	const entry = mockEntry(id);
	if (!entry.draft.provenance) throw new Error(`mock draft ${id} is not ready`);
	for (const { positionId } of positions) addMockRole(entry, positionId);
	if (!entry.draft.provenance) throw new Error(`mock draft ${id} is not ready`);
	const order =
		roles ?? entry.draft.provenance.positions.map((p) => p.positionId);
	const byRole = <T extends { positionId: string }>(positions: T[]) =>
		order.flatMap((r) => positions.filter((p) => p.positionId === r));
	const chars = slots.reduce((n, s) => n + s.text.length, 0);
	const findings = entry.draft.findings.filter((f) => f.check !== "page_count");
	if (chars > MOCK_TWO_PAGES_CHARS)
		findings.push({
			check: "page_count",
			severity: "block",
			message: "the Draft is 2 pages; the base CV is 1",
		});
	const replaced = replaceProvenance(entry.draft.provenance, positions);
	const provenance = { ...replaced, positions: byRole(replaced.positions) };
	const { content } = entry.draft;
	const edited =
		content && applySlots(replaceContent(content, positions), replaced, slots);
	entry.draft = {
		...entry.draft,
		findings,
		provenance,
		content: edited && {
			...edited,
			positions: byRole(edited.positions),
			...(skills && {
				skills: skills.flatMap((g) => g.items),
				skillGroups: skills,
			}),
		},
	};
	return entry.draft;
}

function addMockRole(entry: ReturnType<typeof mockEntry>, positionId: string) {
	const { provenance, content } = entry.draft;
	const position = getExperience().find((p) => p.id === positionId);
	if (
		!provenance ||
		!position ||
		provenance.positions.some((p) => p.positionId === positionId)
	)
		return;
	entry.draft = {
		...entry.draft,
		provenance: {
			...provenance,
			positions: [
				...provenance.positions,
				{
					positionId,
					employer: position.employer,
					title: position.title,
					bullets: [],
				},
			],
		},
		content: content && {
			...content,
			positions: [
				...content.positions,
				{ positionId, headingText: "", linked: true, bullets: [] },
			],
		},
	};
}

function replaceContent(
	content: DraftContent,
	positions: DraftPositionEdit[],
): DraftContent {
	return {
		...content,
		positions: content.positions.map((p) => {
			const replaced = positions.find((r) => r.positionId === p.positionId);
			return replaced ? { ...p, ...replaced } : p;
		}),
	};
}

function replaceProvenance(
	provenance: DraftProvenance,
	positions: DraftPositionEdit[],
): DraftProvenance {
	return {
		...provenance,
		positions: provenance.positions.map((p) => {
			const replaced = positions.find((r) => r.positionId === p.positionId);
			if (!replaced) return p;
			return {
				...p,
				bullets: replaced.bullets.map((b, i) => ({
					slotId: p.bullets[i]?.slotId ?? `s-${p.positionId}-${i}`,
					achievements: getExperience()
						.flatMap((position) => position.achievements)
						.filter((a) => b.achievementIds.includes(a.id))
						.map(({ id, positionId, text }) => ({ id, positionId, text })),
				})),
			};
		}),
	};
}

function applySlots(
	content: DraftContent,
	provenance: DraftProvenance,
	slots: SlotEdit[],
): DraftContent {
	const text = new Map(slots.map((s) => [s.slotId, s.text]));
	return {
		...content,
		profile: text.get(provenance.profile?.slotId ?? "") ?? content.profile,
		positions: content.positions.map((position) => {
			const bullets = provenance.positions.find(
				(p) => p.positionId === position.positionId,
			)?.bullets;
			return {
				...position,
				bullets: position.bullets.map((bullet, i) => ({
					...bullet,
					text: text.get(bullets?.[i]?.slotId ?? "") ?? bullet.text,
				})),
			};
		}),
	};
}

function seedDraft(id: string, outcome: Draft["outcome"], jobId?: string) {
	mockDrafts.set(id, {
		polls: 0,
		draft: {
			id,
			jobId: jobId ?? getJobs().at(-1)?.ID ?? "",
			mode: "generate",
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
			skillsEditable: true,
			insights: null,
			baseDocId: "mock-doc",
			baseTabId: "mock-tab",
			...mockContent(),
		},
	});
}

function seedUnsettled(id: string, status: Draft["status"], lastError = "") {
	seedDraft(id, null);
	const entry = mockEntry(id);
	entry.draft = {
		...entry.draft,
		jobId: getJobs().at(-2)?.ID ?? "",
		status,
		lastError,
		draftDocUrl: null,
		findings: [],
		provenance: null,
		content: null,
		base: null,
	};
}

seedDraft("draft-ready", null);
seedUnsettled("draft-pending", "pending");
seedUnsettled("draft-failed", "failed", "the model timed out");
seedDraft("draft-kept", "kept", seed.applications.at(-1)?.JobID);

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
	skillLine: null,
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

const mockSkillRow = (
	skillLine: number,
	label: string,
	value: string,
): LayoutBlock =>
	mockBlock({
		section: "skills",
		skillLine,
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
			mockSkillRow(0, "Languages", "Go, SQL, TypeScript"),
			mockSkillRow(1, "Tools", "Kubernetes, Postgres, Terraform"),
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
