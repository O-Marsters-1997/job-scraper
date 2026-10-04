const seen = new Set<string>();

export function isSeen(jobId: string): boolean {
	return seen.has(jobId);
}

export function markJobsSeen(jobIds: string[], value: boolean): void {
	for (const id of jobIds) {
		if (value) seen.add(id);
		else seen.delete(id);
	}
}
