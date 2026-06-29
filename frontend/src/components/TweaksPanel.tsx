import { cva } from "class-variance-authority";
import type { JSX } from "solid-js";
import { createSignal, For, onCleanup, onMount, Show } from "solid-js";
import { ColorPicker } from "@/components/ui/color-picker";
import { oklchToHex, toOklch } from "@/lib/color";
import {
	CUSTOM_DEFAULTS,
	DEFAULTS,
	type DensityKey,
	FONT_KEYS,
	FONTS,
	type FontKey,
	loadTweaks,
	type RadiusKey,
	type SidebarWidthKey,
	type SizeKey,
	saveTweaks,
	THEME_KEYS,
	THEME_VAR_GROUPS,
	THEMES,
	type ThemeKey,
	type Tweaks,
} from "@/lib/tweaks";
import { applyAll, applyTheme } from "@/lib/tweaks.apply";
import { cn } from "@/lib/utils";

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

const selectableButtonVariants = cva(
	"bg-transparent transition-[border-color,background]",
	{
		variants: {
			selected: {
				true: "border-primary bg-accent-subtle",
				false: "border-border hover:border-border-strong",
			},
		},
		defaultVariants: { selected: false },
	},
);

export default function TweaksPanel() {
	let panelRef: HTMLDivElement | undefined;

	const [open, setOpen] = createSignal(false);
	const [view, setView] = createSignal<"main" | "custom">("main");
	const [tweaks, setTweaks] = createSignal<Tweaks>(DEFAULTS);
	// Draft colours while the custom editor is open — not yet persisted
	const [draft, setDraft] =
		createSignal<Record<string, string>>(CUSTOM_DEFAULTS);

	onMount(() => {
		const loaded = loadTweaks();
		setTweaks(loaded);
		applyAll(loaded);

		// Close panel when clicking outside (but not inside picker popovers)
		function handlePointerDown(e: PointerEvent) {
			if (!open()) return;
			const target = e.target as Element | null;
			if (!target) return;
			// Kobalte Popover portals render in <body>; let clicks inside them pass
			if (target.closest("[data-tweaks-portal]")) return;
			if (panelRef && !panelRef.contains(target)) setOpen(false);
		}

		document.addEventListener("pointerdown", handlePointerDown);
		onCleanup(() =>
			document.removeEventListener("pointerdown", handlePointerDown),
		);
	});

	function update<K extends keyof Tweaks>(key: K, value: Tweaks[K]) {
		const next = { ...tweaks(), [key]: value };
		setTweaks(next);
		saveTweaks(next);
		applyAll(next);
	}

	function reset() {
		setTweaks({ ...DEFAULTS });
		saveTweaks(DEFAULTS);
		applyAll(DEFAULTS);
		setView("main");
	}

	function openCustomEditor() {
		const colors = tweaks().customColors ?? CUSTOM_DEFAULTS;
		setDraft({ ...colors });
		// Live-preview the current custom palette immediately
		applyTheme("custom", colors);
		setView("custom");
	}

	function cancelCustomEditor() {
		// Revert any live preview edits
		applyAll(tweaks());
		setView("main");
	}

	function saveCustomTheme() {
		const next: Tweaks = {
			...tweaks(),
			theme: "custom",
			customColors: draft(),
		};
		setTweaks(next);
		saveTweaks(next);
		applyAll(next);
		setView("main");
	}

	function updateDraftColor(varName: string, value: string) {
		const updated = { ...draft(), [varName]: value };
		setDraft(updated);
		// Live-apply so the user can see the change immediately
		applyTheme("custom", updated);
	}

	const t = () => tweaks();

	// Derive a live swatch for the Custom button from saved customColors.
	// Stored values are OKLCH; convert to hex for inline preview styling.
	const customSwatch = () => {
		const c = t().customColors ?? CUSTOM_DEFAULTS;
		const hex = (key: string, fallback: string) =>
			c[key] ? oklchToHex(c[key]) : fallback;
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
		<div
			ref={panelRef}
			class="fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2"
		>
			<Show when={open()}>
				<div
					class="flex w-72 flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-xl"
					style={{ "max-height": "86vh" }}
				>
					{/* ── Header ── */}
					<div class="flex shrink-0 items-center justify-between border-b border-border px-3.5 py-2.5">
						<Show
							when={view() === "custom"}
							fallback={
								<div class="flex items-center gap-2">
									<span class="text-data font-semibold tracking-tight text-foreground">
										Tweaks
									</span>
									<span class="rounded-full border border-accent-border bg-accent-subtle px-2 py-0.5 text-2xs font-semibold text-accent-text">
										{THEMES[t().theme].name}
									</span>
								</div>
							}
						>
							<div class="flex items-center gap-2">
								<button
									type="button"
									onClick={cancelCustomEditor}
									class="flex size-6 items-center justify-center rounded text-faint transition-colors hover:bg-background hover:text-foreground"
									aria-label="Back to tweaks"
								>
									<svg
										width="14"
										height="14"
										viewBox="0 0 24 24"
										fill="none"
										stroke="currentColor"
										stroke-width="2"
										stroke-linecap="round"
										stroke-linejoin="round"
										aria-hidden="true"
									>
										<polyline points="15 18 9 12 15 6" />
									</svg>
								</button>
								<span class="text-data font-semibold tracking-tight text-foreground">
									Custom theme
								</span>
							</div>
						</Show>
						<button
							type="button"
							onClick={() => {
								if (view() === "custom") cancelCustomEditor();
								setOpen(false);
							}}
							class="flex size-6 items-center justify-center rounded text-faint transition-colors hover:bg-background hover:text-foreground"
							aria-label="Close tweaks"
						>
							<svg
								width="14"
								height="14"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
								stroke-linecap="round"
								stroke-linejoin="round"
								aria-hidden="true"
							>
								<line x1="18" y1="6" x2="6" y2="18" />
								<line x1="6" y1="6" x2="18" y2="18" />
							</svg>
						</button>
					</div>

					{/* ── Body ── */}
					<div
						class="flex-1 overflow-y-auto"
						style={{ "scrollbar-width": "thin" }}
					>
						<Show
							when={view() === "custom"}
							fallback={
								<>
									<ThemeSection
										selected={t().theme}
										swatchFor={swatchFor}
										onSelect={(key) => {
											if (key === "custom") openCustomEditor();
											else update("theme", key);
										}}
									/>
									<TypographySection
										selected={t().font}
										onSelect={(key) => update("font", key)}
									/>
									<SizeSection
										selected={t().size}
										onSelect={(key) => update("size", key)}
									/>
									<SidebarWidthSection
										selected={t().sidebarWidth}
										onSelect={(key) => update("sidebarWidth", key)}
									/>
									<DensitySection
										selected={t().density}
										onSelect={(key) => update("density", key)}
									/>
									<RadiusSection
										selected={t().radius}
										onSelect={(key) => update("radius", key)}
									/>
								</>
							}
						>
							<CustomThemeEditor draft={draft} onUpdate={updateDraftColor} />
						</Show>
					</div>

					{/* ── Footer ── */}
					<div class="flex shrink-0 items-center justify-between border-t border-border px-3.5 py-2.5">
						<Show
							when={view() === "custom"}
							fallback={
								<>
									<button
										type="button"
										onClick={reset}
										class="text-xs text-faint transition-colors hover:text-foreground hover:underline"
									>
										Reset defaults
									</button>
									<span class="font-mono text-2xs text-faint">
										{t().theme} · {FONTS[t().font].name} ·{" "}
										{t().size.toUpperCase()}
									</span>
								</>
							}
						>
							<button
								type="button"
								onClick={cancelCustomEditor}
								class="text-xs text-faint transition-colors hover:text-foreground hover:underline"
							>
								Cancel
							</button>
							<button
								type="button"
								onClick={saveCustomTheme}
								class="rounded-md bg-primary px-3 py-1 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary-hover"
							>
								Save
							</button>
						</Show>
					</div>
				</div>
			</Show>

			{/* Toggle FAB */}
			<button
				type="button"
				onClick={() => setOpen((o) => !o)}
				class={cn(
					"flex size-9 items-center justify-center rounded-full border shadow-lg transition-colors",
					open()
						? "border-primary bg-primary text-primary-foreground"
						: "border-border bg-surface text-muted hover:border-border-strong hover:text-foreground",
				)}
				title="Tweaks"
				aria-label="Open tweaks panel"
			>
				{/* Lucide SlidersHorizontal */}
				<svg
					width="16"
					height="16"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
					aria-hidden="true"
				>
					<line x1="21" y1="4" x2="7" y2="4" />
					<line x1="3" y1="4" x2="7" y2="4" />
					<line x1="21" y1="12" x2="11" y2="12" />
					<line x1="7" y1="12" x2="3" y2="12" />
					<line x1="21" y1="20" x2="15" y2="20" />
					<line x1="11" y1="20" x2="3" y2="20" />
					<circle cx="7" cy="4" r="2" />
					<circle cx="11" cy="12" r="2" />
					<circle cx="15" cy="20" r="2" />
				</svg>
			</button>
		</div>
	);
}

// ── Sub-panels ─────────────────────────────────────────────────────────────────

function ThemeSection(props: {
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
						const sw = () => props.swatchFor(key as ThemeKey);
						const selected = () => props.selected === key;
						return (
							<button
								type="button"
								onClick={() => props.onSelect(key as ThemeKey)}
								aria-pressed={selected()}
								class={themeButtonVariants({ selected: selected() })}
							>
								{/* Mini-app preview: sidebar + content card + accent */}
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
										<svg
											width="10"
											height="10"
											viewBox="0 0 24 24"
											fill="none"
											stroke="currentColor"
											stroke-width="3"
											stroke-linecap="round"
											stroke-linejoin="round"
											aria-hidden="true"
										>
											<polyline points="20 6 9 17 4 12" />
										</svg>
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

function TypographySection(props: {
	selected: FontKey;
	onSelect: (key: FontKey) => void;
}) {
	return (
		<Section label="Typography" hint="UI + data font pair">
			<div class="grid grid-cols-3 gap-1.5">
				<For each={FONT_KEYS}>
					{(key) => (
						<button
							type="button"
							onClick={() => props.onSelect(key as FontKey)}
							style={{ "font-family": FONTS[key].ui }}
							class={cn(
								"flex flex-col items-center gap-0.5 rounded-md border-[1.5px] px-1 py-2",
								selectableButtonVariants({ selected: props.selected === key }),
							)}
						>
							<span class="block text-xl font-semibold leading-tight text-foreground">
								Ag
							</span>
							<span
								class={cn(
									"block text-2xs font-semibold",
									props.selected === key ? "text-accent-text" : "text-faint",
								)}
								style={{ "font-family": "'Plus Jakarta Sans', sans-serif" }}
							>
								{FONTS[key].name}
							</span>
						</button>
					)}
				</For>
			</div>
		</Section>
	);
}

function SizeSection(props: {
	selected: SizeKey;
	onSelect: (key: SizeKey) => void;
}) {
	const OPTIONS = [
		{ key: "xs" as SizeKey, label: "XS", px: "12px" },
		{ key: "sm" as SizeKey, label: "S", px: "14px" },
		{ key: "md" as SizeKey, label: "M", px: "17px" },
		{ key: "lg" as SizeKey, label: "L", px: "20px" },
	];
	return (
		<Section label="Size scale" hint="base → headings proportional">
			<div class="flex gap-1.5">
				<For each={OPTIONS}>
					{(opt) => (
						<button
							type="button"
							onClick={() => props.onSelect(opt.key)}
							class={cn(
								"flex flex-1 flex-col items-center gap-0.5 rounded-md border-[1.5px] px-1 py-1.5",
								selectableButtonVariants({
									selected: props.selected === opt.key,
								}),
							)}
						>
							<span
								class="block font-bold leading-none text-foreground"
								style={{ "font-size": opt.px }}
							>
								Aa
							</span>
							<span
								class={cn(
									"block text-2xs font-semibold",
									props.selected === opt.key
										? "text-accent-text"
										: "text-faint",
								)}
							>
								{opt.label}
							</span>
						</button>
					)}
				</For>
			</div>
		</Section>
	);
}

function SidebarWidthSection(props: {
	selected: SidebarWidthKey;
	onSelect: (key: SidebarWidthKey) => void;
}) {
	return (
		<Section label="Sidebar width">
			<SegControl
				options={[
					{ value: "narrow", label: "Narrow" },
					{ value: "default", label: "Default" },
					{ value: "wide", label: "Wide" },
				]}
				value={props.selected}
				onChange={(v) => props.onSelect(v as SidebarWidthKey)}
			/>
		</Section>
	);
}

function DensitySection(props: {
	selected: DensityKey;
	onSelect: (key: DensityKey) => void;
}) {
	return (
		<Section label="Row density">
			<SegControl
				options={[
					{ value: "compact", label: "Compact" },
					{ value: "default", label: "Default" },
					{ value: "spacious", label: "Spacious" },
				]}
				value={props.selected}
				onChange={(v) => props.onSelect(v as DensityKey)}
			/>
		</Section>
	);
}

function RadiusSection(props: {
	selected: RadiusKey;
	onSelect: (key: RadiusKey) => void;
}) {
	return (
		<Section label="Corner radius">
			<SegControl
				options={[
					{ value: "sharp", label: "Sharp" },
					{ value: "default", label: "Default" },
					{ value: "round", label: "Round" },
				]}
				value={props.selected}
				onChange={(v) => props.onSelect(v as RadiusKey)}
			/>
		</Section>
	);
}

function CustomThemeEditor(props: {
	draft: () => Record<string, string>;
	onUpdate: (varName: string, value: string) => void;
}) {
	return (
		<For each={THEME_VAR_GROUPS}>
			{(group) => (
				<Section label={group.label}>
					<div class="flex flex-col gap-1.5">
						<For each={group.vars}>
							{(item) => (
								<div class="flex items-center justify-between gap-2">
									<span class="text-xs text-muted">{item.label}</span>
									<ColorPicker
										value={oklchToHex(props.draft()[item.key] ?? "#000000")}
										onChange={(v) => props.onUpdate(item.key, toOklch(v))}
										label={item.label}
									/>
								</div>
							)}
						</For>
					</div>
				</Section>
			)}
		</For>
	);
}

// ── Shared layout primitives ───────────────────────────────────────────────────

interface SectionProps {
	label: string;
	hint?: string;
	children: JSX.Element;
}

function Section(props: SectionProps) {
	return (
		<div class="border-t border-border px-3.5 py-2.5">
			<p class="mb-2 flex items-baseline gap-1.5 text-2xs font-bold uppercase tracking-[0.08em] text-faint">
				{props.label}
				<Show when={props.hint}>
					<span class="text-2xs font-normal normal-case tracking-normal opacity-65">
						{props.hint}
					</span>
				</Show>
			</p>
			{props.children}
		</div>
	);
}

interface SegOption {
	value: string;
	label: string;
}

interface SegControlProps {
	options: SegOption[];
	value: string;
	onChange: (value: string) => void;
}

function SegControl(props: SegControlProps) {
	return (
		<div class="flex overflow-hidden rounded-md border border-border">
			<For each={props.options}>
				{(opt, i) => (
					<button
						type="button"
						onClick={() => props.onChange(opt.value)}
						class={cn(
							"flex-1 py-1.5 text-xs font-medium transition-colors",
							i() < props.options.length - 1 && "border-r border-border",
							props.value === opt.value
								? "bg-foreground text-surface"
								: "text-muted hover:bg-background hover:text-foreground",
						)}
					>
						{opt.label}
					</button>
				)}
			</For>
		</div>
	);
}
