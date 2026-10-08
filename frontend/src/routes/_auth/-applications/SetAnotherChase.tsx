import { createSignal, Show } from "solid-js";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSetChase } from "../../../hooks/useApplications";

export function SetAnotherChase(props: {
	applicationId: string;
	title: string;
	onDone: () => void;
}) {
	const [picking, setPicking] = createSignal(false);
	const [draft, setDraft] = createSignal("");
	const setChase = useSetChase();

	const save = () => {
		if (!draft()) return;
		setChase.mutate(
			{ id: props.applicationId, chaseBy: draft() },
			{ onSuccess: props.onDone },
		);
	};

	return (
		<div class="mb-4 flex flex-wrap items-center gap-3 rounded-xl border border-border bg-surface px-4 py-3">
			<p class="text-sm text-foreground">Chased {props.title}. Set another?</p>
			<Show
				when={picking()}
				fallback={
					<div class="flex gap-2">
						<Button size="sm" onClick={() => setPicking(true)}>
							Yes
						</Button>
						<Button variant="outline" size="sm" onClick={props.onDone}>
							No
						</Button>
					</div>
				}
			>
				<div class="flex items-center gap-2">
					<Input
						type="date"
						aria-label="Chase by"
						value={draft()}
						onInput={(e) => setDraft(e.currentTarget.value)}
					/>
					<Button
						size="sm"
						disabled={!draft() || setChase.isPending}
						onClick={save}
					>
						Save
					</Button>
					<Button variant="outline" size="sm" onClick={props.onDone}>
						Cancel
					</Button>
				</div>
			</Show>
		</div>
	);
}
