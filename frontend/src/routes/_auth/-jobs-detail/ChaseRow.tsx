import { createSignal, Match, Show, Switch } from "solid-js";
import { FactRow } from "@/components/FactRow";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { chaseDateKey, formatChaseDate, isChaseOverdue } from "@/lib/chase";
import { cn } from "@/lib/utils";
import { useClearChase, useSetChase } from "../../../hooks/useApplications";

type Mode = "idle" | "picking" | "another";

export function ChaseRow(props: {
	applicationId: string;
	chaseBy: string | null;
}) {
	const [mode, setMode] = createSignal<Mode>("idle");
	const [draft, setDraft] = createSignal("");
	const setChase = useSetChase();
	const clearChase = useClearChase();

	const openPicker = () => {
		setDraft(props.chaseBy ? chaseDateKey(props.chaseBy) : "");
		setMode("picking");
	};

	const save = () => {
		if (!draft()) return;
		setChase.mutate(
			{ id: props.applicationId, chaseBy: draft() },
			{ onSuccess: () => setMode("idle") },
		);
	};

	const markChased = () => {
		clearChase.mutate(props.applicationId, {
			onSuccess: () => setMode("another"),
		});
	};

	return (
		<Switch>
			<Match when={mode() === "picking"}>
				<div class="flex flex-col gap-2">
					<Input
						type="date"
						aria-label="Chase by"
						value={draft()}
						onInput={(e) => setDraft(e.currentTarget.value)}
					/>
					<div class="flex gap-2">
						<Button
							size="sm"
							class="flex-1"
							disabled={!draft() || setChase.isPending}
							onClick={save}
						>
							Save
						</Button>
						<Button
							variant="outline"
							size="sm"
							class="flex-1"
							onClick={() => setMode("idle")}
						>
							Cancel
						</Button>
					</div>
				</div>
			</Match>
			<Match when={mode() === "another"}>
				<div class="flex flex-col gap-2">
					<p class="text-xs text-muted">Chased. Set another?</p>
					<div class="flex gap-2">
						<Button size="sm" class="flex-1" onClick={openPicker}>
							Yes
						</Button>
						<Button
							variant="outline"
							size="sm"
							class="flex-1"
							onClick={() => setMode("idle")}
						>
							No
						</Button>
					</div>
				</div>
			</Match>
			<Match when={mode() === "idle"}>
				<Show
					when={props.chaseBy}
					fallback={
						<Button
							variant="outline"
							size="sm"
							class="w-full"
							onClick={openPicker}
						>
							Set chase
						</Button>
					}
				>
					{(chaseBy) => (
						<div class="flex flex-col gap-2">
							<FactRow label="Chase by">
								<span
									class={cn(
										"font-mono tabular-nums",
										isChaseOverdue(chaseBy(), new Date())
											? "text-destructive-strong"
											: "text-foreground",
									)}
								>
									{formatChaseDate(chaseBy())}
								</span>
							</FactRow>
							<div class="flex gap-2">
								<Button
									size="sm"
									class="flex-1"
									disabled={clearChase.isPending}
									onClick={markChased}
								>
									Chased
								</Button>
								<Button
									variant="outline"
									size="sm"
									class="flex-1"
									onClick={openPicker}
								>
									Edit chase
								</Button>
							</div>
						</div>
					)}
				</Show>
			</Match>
		</Switch>
	);
}
