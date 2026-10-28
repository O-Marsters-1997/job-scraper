import { createSignal, For } from "solid-js";
import { QueryBoundary } from "@/components/QueryBoundary";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
	useLetterSkeleton,
	useSaveLetterSkeleton,
} from "../../hooks/useLetterSkeleton";
import type { LetterSkeleton } from "../../types/coverLetter";

export function LetterSkeletonEditor() {
	const query = useLetterSkeleton();
	return (
		<QueryBoundary query={query} fallbackRows={3}>
			{(skeleton) => <SkeletonForm initial={skeleton()} />}
		</QueryBoundary>
	);
}

function SkeletonForm(props: { initial: LetterSkeleton }) {
	const save = useSaveLetterSkeleton();
	const [intro, setIntro] = createSignal(props.initial.intro);
	const [conclusion, setConclusion] = createSignal(props.initial.conclusion);
	const [sections, setSections] = createSignal(props.initial.sections);

	const setSection = (i: number, value: string) =>
		setSections((s) => s.map((v, j) => (j === i ? value : v)));
	const move = (i: number, by: number) =>
		setSections((s) => {
			const next = [...s];
			const [item] = next.splice(i, 1);
			next.splice(i + by, 0, item as string);
			return next;
		});

	const submit = (e: SubmitEvent) => {
		e.preventDefault();
		save.mutate({
			intro: intro(),
			conclusion: conclusion(),
			sections: sections()
				.map((s) => s.trim())
				.filter((s) => s !== ""),
		});
	};

	return (
		<form onSubmit={submit} class="grid max-w-2xl gap-4">
			<div>
				<Label for="skeleton-intro">Intro</Label>
				<Textarea
					id="skeleton-intro"
					rows={3}
					placeholder="Dear ___, I'm writing about the ___ role at ___."
					value={intro()}
					onInput={(e) => setIntro(e.currentTarget.value)}
				/>
			</div>
			<div class="grid gap-2">
				<Label>Body sections</Label>
				<For each={sections()}>
					{(section, i) => (
						<div class="flex gap-2">
							<Input
								class="flex-1"
								aria-label={`Section ${i() + 1} purpose`}
								placeholder="What experience makes me a good fit"
								value={section}
								onInput={(e) => setSection(i(), e.currentTarget.value)}
							/>
							<Button
								type="button"
								variant="outline"
								aria-label={`Move section ${i() + 1} up`}
								disabled={i() === 0}
								onClick={() => move(i(), -1)}
							>
								Up
							</Button>
							<Button
								type="button"
								variant="outline"
								aria-label={`Move section ${i() + 1} down`}
								disabled={i() === sections().length - 1}
								onClick={() => move(i(), 1)}
							>
								Down
							</Button>
							<Button
								type="button"
								variant="outline"
								aria-label={`Remove section ${i() + 1}`}
								onClick={() =>
									setSections((s) => s.filter((_, j) => j !== i()))
								}
							>
								Remove
							</Button>
						</div>
					)}
				</For>
				<div>
					<Button
						type="button"
						variant="outline"
						onClick={() => setSections((s) => [...s, ""])}
					>
						Add section
					</Button>
				</div>
			</div>
			<div>
				<Label for="skeleton-conclusion">Conclusion</Label>
				<Textarea
					id="skeleton-conclusion"
					rows={3}
					placeholder="Thank you for considering me. I'd love to talk about ___."
					value={conclusion()}
					onInput={(e) => setConclusion(e.currentTarget.value)}
				/>
			</div>
			<div>
				<Button type="submit" loading={save.isPending}>
					Save skeleton
				</Button>
			</div>
		</form>
	);
}
