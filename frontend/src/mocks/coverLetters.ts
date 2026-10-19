import type { CoverLetter, ScaffoldInput } from "../types/coverLetter";

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
	};
	coverLetters.push(letter);
	return letter;
}
