import { For, type JSX, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import { cn } from "@/lib/utils";

export function Field(props: {
	label: string;
	for: string;
	hint?: string;
	children: JSX.Element;
}) {
	return (
		<div>
			<label for={props.for} class="block text-sm font-medium text-foreground">
				{props.label}
			</label>
			<Show when={props.hint}>
				<p class="mt-0.5 text-xs text-faint">{props.hint}</p>
			</Show>
			<div class="mt-2">{props.children}</div>
		</div>
	);
}

export type Choice = { label: string; tone: string };

export function ChoiceGroup(props: {
	legend: string;
	hint?: string;
	options: { id: string; label: string }[];
	choiceOf: (id: string) => Choice | undefined;
	cycle: (id: string) => void;
}) {
	return (
		<fieldset>
			<legend class="text-sm font-medium text-foreground">
				{props.legend}
			</legend>
			<Show when={props.hint}>
				<p class="mt-0.5 text-xs text-faint">{props.hint}</p>
			</Show>
			<div class="mt-2 flex flex-wrap gap-2">
				<For each={props.options}>
					{(o) => {
						const choice = () => props.choiceOf(o.id);
						return (
							<button
								type="button"
								aria-pressed={choice() !== undefined}
								onClick={() => props.cycle(o.id)}
								class={cn(
									"inline-flex h-8 items-center gap-2 rounded-md border px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
									choice()
										? cn("font-medium", choice()?.tone)
										: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
								)}
							>
								<span
									aria-hidden="true"
									class={cn(
										"grid size-3.5 place-items-center rounded-sm border",
										choice()
											? "border-current bg-current"
											: "border-border-strong bg-surface",
									)}
								>
									<Show when={choice()}>
										<Icon
											name="check"
											size={10}
											strokeWidth={3.5}
											class="text-surface"
										/>
									</Show>
								</span>
								{o.label}
								<Show when={choice()}>
									{(c) => (
										<span class="text-xs font-normal opacity-80">
											{c().label}
										</span>
									)}
								</Show>
							</button>
						);
					}}
				</For>
			</div>
		</fieldset>
	);
}
