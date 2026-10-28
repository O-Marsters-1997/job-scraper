import type {
	CoverLetter,
	LetterCheck,
	ScaffoldInput,
} from "../types/coverLetter";

const coverLetters: CoverLetter[] = [];

export function getCoverLetters(jobId: string): CoverLetter[] {
	return coverLetters.filter((c) => c.jobId === jobId).reverse();
}

export function createCoverLetter(input: ScaffoldInput): CoverLetter {
	const letter: CoverLetter = {
		id: `cover-letter-${coverLetters.length + 1}`,
		jobId: input.jobId,
		sourceDocId: input.docId,
		sourceTabId: input.tabId,
		tabUrl: `https://docs.google.com/document/d/${input.docId}/edit?tab=t.scaffold${coverLetters.length + 1}`,
		achievementIds: input.achievementIds,
		cost: 0.01,
		createdAt: new Date().toISOString(),
		doneAt: null,
	};
	coverLetters.push(letter);
	return letter;
}

export function checkCoverLetter(_id: string): LetterCheck {
	return {
		words: 212,
		findings: [
			{ kind: "length", text: "212 words, aim for 250 to 400" },
			{ kind: "leftover_gap", text: "[GAP] No Kubernetes yet" },
		],
	};
}

export function markCoverLetterDone(id: string): CoverLetter {
	const letter = coverLetters.find((c) => c.id === id);
	if (!letter) throw new Error("cover letter not found");
	letter.doneAt ??= new Date().toISOString();
	return letter;
}
