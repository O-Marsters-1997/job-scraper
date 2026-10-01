import { createEffect, createSignal, For, Show } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { allConfirmed, toMappings } from "@/lib/tailoring";
import { useExperience } from "../../../hooks/useExperience";
import {
	type CVRef,
	useHeadings,
	useSaveHeadings,
} from "../../../hooks/useTailoring";
import type { Position } from "../../../types/experience";
import type { CVHeading } from "../../../types/tailoring";

export function HeadingsStep(props: {
	cv: () => CVRef | undefined;
	onDone: (skipped: boolean) => void;
	onBack: () => void;
}) {
	const headings = useHeadings(() => props.cv());
	const positions = useExperience();
	const save = useSaveHeadings();
	const [chosen, setChosen] = createSignal<Record<string, string | null>>({});

	createEffect(() => {
		const data = headings.data;
		if (data && allConfirmed(data)) props.onDone(true);
	});

	const submit = (data: CVHeading[]) => {
		const ref = props.cv();
		if (!ref) return;
		save.mutate(
			{ ...ref, mappings: toMappings(data, chosen()) },
			{ onSuccess: () => props.onDone(false) },
		);
	};

	const valueFor = (h: CVHeading) =>
		h.text in chosen() ? (chosen()[h.text] ?? "") : (h.positionId ?? "");

	return (
		<QueryBoundary query={headings}>
			{(data) => (
				<Card class="p-5">
					<p class="mb-4 text-sm text-muted">
						Match each role heading in this CV to a Position. Choose None to
						leave a section untouched.
					</p>
					<div class="space-y-3">
						<For each={data()}>
							{(h) => (
								<label class="grid items-center gap-2 sm:grid-cols-[1fr_260px]">
									<span class="text-sm text-foreground">{h.text}</span>
									<select
										class="h-9 rounded-md border border-border bg-surface px-3 text-sm text-foreground"
										value={valueFor(h)}
										onChange={(e) =>
											setChosen((c) => ({
												...c,
												[h.text]: e.currentTarget.value || null,
											}))
										}
									>
										<option value="">None</option>
										<For each={positions.data ?? []}>
											{(p: Position) => (
												<option value={p.id}>
													{p.title}, {p.employer}
												</option>
											)}
										</For>
									</select>
								</label>
							)}
						</For>
					</div>
					<Show when={save.error}>
						<p class="mt-3 text-sm text-danger">Could not save the mapping.</p>
					</Show>
					<div class="mt-5 flex gap-2">
						<Button variant="outline" onClick={props.onBack}>
							Back
						</Button>
						<Button disabled={save.isPending} onClick={() => submit(data())}>
							Save and continue
						</Button>
					</div>
				</Card>
			)}
		</QueryBoundary>
	);
}
