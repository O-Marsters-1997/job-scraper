import { cva } from "class-variance-authority";
import { For, Show } from "solid-js";
import { Icon } from "@/components/Icon";
import type { ThemeKey } from "@/lib/tweaks";
import { THEME_KEYS, THEMES } from "@/lib/tweaks.themes";
import { Section } from "./primitives";

const themeButtonVariants = cva(
	"group relative overflow-hidden rounded-lg border bg-transparent text-left transition-[transform,box-shadow,border-color] duration-150 ease-out",
	{
		variants: {
			selected: {
				true: "border-primary shadow-[0_0_0_2px_var(--color-primary)]",
				false:
					"border-border hover:-translate-y-0.5 hover:border-border-strong hover:shadow-sm",
			},
		},
		defaultVariants: { selected: false },
	},
);

const themeButtonLabelVariants = cva(
	"block border-t py-1 text-center text-xs font-semibold transition-colors",
	{
		variants: {
			selected: {
				true: "border-primary/30 bg-accent-subtle text-accent-text",
				false:
					"border-border bg-surface text-muted group-hover:text-foreground",
			},
		},
		defaultVariants: { selected: false },
	},
);

export function ThemeSection(props: {
	selected: ThemeKey;
	swatchFor: (key: ThemeKey) => {
		sb: string;
		cv: string;
		ac: string;
		su: string;
		bd: string;
	};
	onSelect: (key: ThemeKey) => void;
}) {
	return (
		<Section label="Theme" hint="full palette preview">
			<div class="grid grid-cols-2 gap-2">
				<For each={THEME_KEYS}>
					{(key) => {
						const sw = () => props.swatchFor(key);
						const selected = () => props.selected === key;
						return (
							<button
								type="button"
								onClick={() => props.onSelect(key)}
								aria-pressed={selected()}
								class={themeButtonVariants({ selected: selected() })}
							>
								<div class="flex h-[3.25rem]">
									<div
										class="flex w-[36%] shrink-0 flex-col justify-center gap-[3px] px-2"
										style={{ background: sw().sb }}
									>
										<span
											class="h-1 w-7 rounded-full"
											style={{ background: sw().ac }}
										/>
										<span class="h-1 w-5 rounded-full bg-white/25" />
										<span class="h-1 w-6 rounded-full bg-white/15" />
									</div>
									<div class="flex-1 p-1.5" style={{ background: sw().cv }}>
										<div
											class="flex h-full w-full flex-col gap-[3px] rounded-[5px] border p-1.5"
											style={{ background: sw().su, "border-color": sw().bd }}
										>
											<span
												class="h-1.5 w-8 rounded-full"
												style={{ background: sw().ac }}
											/>
											<span
												class="h-1 w-full rounded-full"
												style={{ background: sw().bd }}
											/>
											<span
												class="h-1 w-2/3 rounded-full"
												style={{ background: sw().bd }}
											/>
										</div>
									</div>
								</div>
								<Show when={selected()}>
									<span class="absolute right-1.5 top-1.5 flex size-4 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm">
										<Icon name="check" size={10} strokeWidth={3} />
									</span>
								</Show>
								<span
									class={themeButtonLabelVariants({ selected: selected() })}
								>
									{THEMES[key].name}
								</span>
							</button>
						);
					}}
				</For>
			</div>
		</Section>
	);
}
