import { For, type JSX, Show } from "solid-js";
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

export function ChoiceGroup(props: {
	legend: string;
	hint?: string;
	options: { id: string; label: string }[];
	isOn: (id: string) => boolean;
	toggle: (id: string) => void;
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
					{(o) => (
						<button
							type="button"
							aria-pressed={props.isOn(o.id)}
							onClick={() => props.toggle(o.id)}
							class={cn(
								"inline-flex h-8 items-center gap-2 rounded-md border px-3 text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary",
								props.isOn(o.id)
									? "border-accent-border bg-accent-subtle font-medium text-accent-text"
									: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
							)}
						>
							<span
								aria-hidden="true"
								class={cn(
									"grid size-3.5 place-items-center rounded-sm border",
									props.isOn(o.id)
										? "border-accent-text bg-accent-text text-primary-foreground"
										: "border-border-strong bg-surface",
								)}
							>
								<Show when={props.isOn(o.id)}>
									<svg
										aria-hidden="true"
										width="10"
										height="10"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="3.5"
										stroke-linecap="round"
										stroke-linejoin="round"
									>
										<polyline points="20 6 9 17 4 12" />
									</svg>
								</Show>
							</span>
							{o.label}
						</button>
					)}
				</For>
			</div>
		</fieldset>
	);
}
