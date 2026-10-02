import { type Accessor, createSignal } from "solid-js";
import { KeptDraftExistsError, keptDraft } from "@/lib/tailoring";
import type { Draft } from "@/types/tailoring";
import { useDiscardDraft, useJobDrafts, useKeepDraft } from "./useTailoring";

export function createKeepFlow(
	draft: Accessor<Draft>,
	flush: () => Promise<boolean>,
) {
	const keepMutation = useKeepDraft();
	const discardMutation = useDiscardDraft();
	const jobDrafts = useJobDrafts(() => draft().jobId);
	const [blockedByKept, setBlockedByKept] = createSignal(false);

	const otherKept = () => {
		const kept = keptDraft(jobDrafts.data ?? []);
		return kept?.id === draft().id ? undefined : kept;
	};
	const keep = async () => {
		if (!(await flush())) return;
		keepMutation.mutate(draft().id, {
			onSuccess: () => setBlockedByKept(false),
			onError: (err) => setBlockedByKept(err instanceof KeptDraftExistsError),
		});
	};
	const replaceKept = () => {
		const kept = otherKept();
		if (kept) discardMutation.mutate(kept.id, { onSuccess: keep });
	};

	return {
		keep,
		discard: () => discardMutation.mutate(draft().id),
		replaceKept,
		blockedByKept,
		otherKept,
		keepMutation,
		discardMutation,
	};
}

export type KeepFlow = ReturnType<typeof createKeepFlow>;
