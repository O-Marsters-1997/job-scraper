import type { LetterSkeleton } from "../types/coverLetter";

let skeleton: LetterSkeleton = { intro: "", conclusion: "", sections: [] };

export function getLetterSkeleton(): LetterSkeleton {
	return skeleton;
}

export function saveLetterSkeleton(input: LetterSkeleton): LetterSkeleton {
	skeleton = input;
	return skeleton;
}
