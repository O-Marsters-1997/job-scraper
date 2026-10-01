import type {
	CVHeading,
	Draft,
	DraftInput,
	DraftProvenance,
	DraftRef,
	HeadingMapping,
	SlotEdit,
	Suggestion,
} from "@/types/tailoring";
import { KeptDraftExistsError } from "../lib/tailoring";
import { getExperience } from "./experience";

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
		}));
	});
}

const mockDrafts = new Map<string, { draft: Draft; polls: number }>();

const MOCK_DOC_URL = "https://docs.google.com/document/d/mock-draft/edit";

function mockProvenance(): DraftProvenance {
	const experience = getExperience();
	const achievements = experience.flatMap((p) => p.achievements).slice(0, 2);
	return {
		positions: experience.slice(0, 1).map((p) => ({
			positionId: p.id,
			employer: p.employer,
			title: p.title,
			bullets: achievements.map((a, i) => ({
				slotId: `s${i}`,
				segments: [
					{ text: `${a.text} using `, novel: false },
					{ text: "Kubernetes", novel: true },
				],
				achievements: [{ id: a.id, positionId: p.id, text: a.text }],
			})),
		})),
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
			draftDocUrl: null,
			lastError: "",
			createdAt: new Date().toISOString(),
			findings: [],
			provenance: null,
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
	if (entry.polls >= 3 && entry.draft.status !== "ready") {
		entry.draft = {
			...entry.draft,
			status: "ready",
			draftDocUrl: MOCK_DOC_URL,
			provenance: mockProvenance(),
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
	entry.draft = { ...entry.draft, outcome: "kept" };
	return entry.draft;
}

export function discardDraft(id: string): Draft {
	const entry = mockEntry(id);
	entry.draft = { ...entry.draft, outcome: "discarded", draftDocUrl: null };
	return entry.draft;
}

export function saveDraftSlots(id: string, slots: SlotEdit[]): Draft {
	const entry = mockEntry(id);
	const provenance = entry.draft.provenance;
	if (!provenance) throw new Error(`mock draft ${id} is not ready`);
	const texts = new Map(slots.map((s) => [s.slotId, s.text]));
	entry.draft = {
		...entry.draft,
		provenance: {
			positions: provenance.positions.map((p) => ({
				...p,
				bullets: p.bullets.map((b) => {
					const text = texts.get(b.slotId);
					return text === undefined
						? b
						: { ...b, segments: [{ text, novel: false }] };
				}),
			})),
		},
	};
	return entry.draft;
}
