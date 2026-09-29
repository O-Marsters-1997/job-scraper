import { createSignal, onCleanup, onMount, untrack } from "solid-js";
import { Icon } from "@/components/Icon";
import { oklchToHex } from "@/lib/color";
import {
	CUSTOM_DEFAULTS,
	DEFAULTS,
	markThemeApplied,
	type ThemeKey,
	type Tweaks,
} from "@/lib/tweaks";
import { applyTheme, applyTweaks } from "@/lib/tweaks.apply";
import { FONTS, THEMES } from "@/lib/tweaks.themes";
import {
	CustomThemeEditor,
	DensitySection,
	RadiusSection,
	SidebarWidthSection,
	SizeSection,
	TypographySection,
} from "./sections";
import { ThemeSection } from "./ThemeSection";

const headerButton =
	"flex size-6 items-center justify-center rounded text-faint transition-colors hover:bg-background hover:text-foreground";

const headerTitle = "text-data font-semibold tracking-tight text-foreground";

const footerLink =
	"text-xs text-faint transition-colors hover:text-foreground hover:underline";

export function TweaksMainView(props: {
	tweaks: () => Tweaks;
	onChange: (next: Tweaks) => void;
	onOpenCustom: () => void;
	onClose: () => void;
}) {
	function update<K extends keyof Tweaks>(key: K, value: Tweaks[K]) {
		props.onChange({ ...props.tweaks(), [key]: value });
	}

	const customSwatch = () => {
		const colors = props.tweaks().customColors ?? CUSTOM_DEFAULTS;
		const hex = (key: string, fallback: string) =>
			colors[key] ? oklchToHex(colors[key]) : fallback;
		return {
			sb: hex("--color-sidebar", "oklch(0.23 0.055 285)"),
			cv: hex("--color-background", "oklch(0.98 0.006 285)"),
			ac: hex("--color-primary", "oklch(0.55 0.18 285)"),
			su: hex("--color-surface", "oklch(1 0 0)"),
			bd: hex("--color-border", "oklch(0.93 0.01 290)"),
		};
	};

	const swatchFor = (key: ThemeKey) =>
		key === "custom" ? customSwatch() : THEMES[key].swatch;

	return (
		<>
			<div class="flex shrink-0 items-center justify-between border-b border-border px-3.5 py-2.5">
				<div class="flex items-center gap-2">
					<span class={headerTitle}>Tweaks</span>
					<span class="rounded-full border border-accent-border bg-accent-subtle px-2 py-0.5 text-2xs font-semibold text-accent-text">
						{THEMES[props.tweaks().theme].name}
					</span>
				</div>
				<button
					type="button"
					onClick={() => props.onClose()}
					class={headerButton}
					aria-label="Close tweaks"
				>
					<Icon name="x" size={14} />
				</button>
			</div>

			<div class="flex-1 overflow-y-auto" style={{ "scrollbar-width": "thin" }}>
				<ThemeSection
					selected={props.tweaks().theme}
					swatchFor={swatchFor}
					onSelect={(key) => {
						if (key === "custom") props.onOpenCustom();
						else update("theme", key);
					}}
				/>
				<TypographySection
					selected={props.tweaks().font}
					onSelect={(key) => update("font", key)}
				/>
				<SizeSection
					selected={props.tweaks().size}
					onSelect={(key) => update("size", key)}
				/>
				<SidebarWidthSection
					selected={props.tweaks().sidebarWidth}
					onSelect={(key) => update("sidebarWidth", key)}
				/>
				<DensitySection
					selected={props.tweaks().density}
					onSelect={(key) => update("density", key)}
				/>
				<RadiusSection
					selected={props.tweaks().radius}
					onSelect={(key) => update("radius", key)}
				/>
			</div>

			<div class="flex shrink-0 items-center justify-between border-t border-border px-3.5 py-2.5">
				<button
					type="button"
					onClick={() => props.onChange({ ...DEFAULTS })}
					class={footerLink}
				>
					Reset defaults
				</button>
				<span class="font-mono text-2xs text-faint">
					{props.tweaks().theme} · {FONTS[props.tweaks().font].name} ·{" "}
					{props.tweaks().size.toUpperCase()}
				</span>
			</div>
		</>
	);
}

export function CustomThemeView(props: {
	tweaks: () => Tweaks;
	onSave: (next: Tweaks) => void;
	onCancel: () => void;
	onClose: () => void;
}) {
	const [draft, setDraft] = createSignal<Record<string, string>>({
		...untrack(() => props.tweaks().customColors ?? CUSTOM_DEFAULTS),
	});

	onMount(() => {
		applyTheme("custom", draft());
		markThemeApplied();
	});
	onCleanup(() => applyTweaks(props.tweaks()));

	function updateDraftColor(varName: string, value: string) {
		const updated = { ...draft(), [varName]: value };
		setDraft(updated);
		applyTheme("custom", updated);
		markThemeApplied();
	}

	return (
		<>
			<div class="flex shrink-0 items-center justify-between border-b border-border px-3.5 py-2.5">
				<div class="flex items-center gap-2">
					<button
						type="button"
						onClick={() => props.onCancel()}
						class={headerButton}
						aria-label="Back to tweaks"
					>
						<Icon name="chevronLeft" size={14} />
					</button>
					<span class={headerTitle}>Custom theme</span>
				</div>
				<button
					type="button"
					onClick={() => props.onClose()}
					class={headerButton}
					aria-label="Close tweaks"
				>
					<Icon name="x" size={14} />
				</button>
			</div>

			<div class="flex-1 overflow-y-auto" style={{ "scrollbar-width": "thin" }}>
				<CustomThemeEditor draft={draft} onUpdate={updateDraftColor} />
			</div>

			<div class="flex shrink-0 items-center justify-between border-t border-border px-3.5 py-2.5">
				<button
					type="button"
					onClick={() => props.onCancel()}
					class={footerLink}
				>
					Cancel
				</button>
				<button
					type="button"
					onClick={() =>
						props.onSave({
							...props.tweaks(),
							theme: "custom",
							customColors: draft(),
						})
					}
					class="rounded-md bg-primary px-3 py-1 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary-hover"
				>
					Save
				</button>
			</div>
		</>
	);
}
