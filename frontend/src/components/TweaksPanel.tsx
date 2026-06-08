import { createSignal, For, onMount, Show } from "solid-js";
import type { JSX } from "solid-js";
import { cn } from "@/lib/utils";
import {
	type DensityKey,
	type FontKey,
	type RadiusKey,
	type SidebarWidthKey,
	type SizeKey,
	type ThemeKey,
	type Tweaks,
	DEFAULTS,
	FONT_KEYS,
	FONTS,
	THEME_KEYS,
	THEMES,
	applyAll,
	loadTweaks,
	saveTweaks,
} from "@/lib/tweaks";

export default function TweaksPanel() {
	const [open, setOpen] = createSignal(false);
	const [tweaks, setTweaks] = createSignal<Tweaks>(DEFAULTS);

	onMount(() => {
		const loaded = loadTweaks();
		setTweaks(loaded);
		applyAll(loaded);
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
	}

	const t = () => tweaks();

	return (
		<div class="fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2">
			<Show when={open()}>
				<div class="flex w-72 flex-col overflow-hidden rounded-xl border border-border bg-surface shadow-xl" style={{ "max-height": "86vh" }}>
					{/* Header */}
					<div class="flex shrink-0 items-center justify-between border-b border-border px-3.5 py-2.5">
						<div class="flex items-center gap-2">
							<span class="text-[13px] font-semibold tracking-tight text-foreground">Tweaks</span>
							<span class="rounded-full border border-accent-border bg-accent-subtle px-2 py-0.5 text-[10px] font-semibold text-accent-text">
								{THEMES[t().theme].name}
							</span>
						</div>
						<button
							type="button"
							onClick={() => setOpen(false)}
							class="flex size-6 items-center justify-center rounded text-faint transition-colors hover:bg-background hover:text-foreground"
							aria-label="Close tweaks"
						>
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<line x1="18" y1="6" x2="6" y2="18" />
								<line x1="6" y1="6" x2="18" y2="18" />
							</svg>
						</button>
					</div>

					{/* Scrollable body */}
					<div class="flex-1 overflow-y-auto" style={{ "scrollbar-width": "thin" }}>
						{/* Theme */}
						<Section label="Theme" hint="colour + sidebar">
							<div class="grid grid-cols-3 gap-1.5">
								<For each={THEME_KEYS}>
									{(key) => (
										<button
											type="button"
											onClick={() => update("theme", key as ThemeKey)}
											class={cn(
												"overflow-hidden rounded-lg border-[1.5px] bg-transparent text-left transition-[border-color,transform]",
												t().theme === key
													? "border-primary"
													: "border-border hover:-translate-y-px hover:border-border-strong",
											)}
										>
											{/* Swatch preview */}
											<div class="flex h-10">
												<div class="w-[30%] shrink-0" style={{ background: THEMES[key].swatch.sb }} />
												<div class="relative flex-1" style={{ background: THEMES[key].swatch.cv }}>
													<div
														class="absolute bottom-1 right-1 size-2.5 rounded-full"
														style={{ background: THEMES[key].swatch.ac }}
													/>
												</div>
											</div>
											<span
												class={cn(
													"block py-1 text-center text-[10.5px] font-semibold",
													t().theme === key ? "text-accent-text" : "text-muted",
												)}
											>
												{THEMES[key].name}
											</span>
										</button>
									)}
								</For>
							</div>
						</Section>

						{/* Typography */}
						<Section label="Typography" hint="UI + data font pair">
							<div class="grid grid-cols-3 gap-1.5">
								<For each={FONT_KEYS}>
									{(key) => (
										<button
											type="button"
											onClick={() => update("font", key as FontKey)}
											style={{ "font-family": FONTS[key].ui }}
											class={cn(
												"flex flex-col items-center gap-0.5 rounded-md border-[1.5px] bg-transparent px-1 py-2 transition-[border-color,background]",
												t().font === key
													? "border-primary bg-accent-subtle"
													: "border-border hover:border-border-strong",
											)}
										>
											<span
												class={cn(
													"block text-[21px] font-semibold leading-tight",
													t().font === key ? "text-foreground" : "text-foreground",
												)}
											>
												Ag
											</span>
											<span
												class={cn(
													"block text-[9.5px] font-semibold",
													t().font === key ? "text-accent-text" : "text-faint",
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

						{/* Size scale */}
						<Section label="Size scale" hint="base → headings proportional">
							<div class="flex gap-1.5">
								<For each={[
									{ key: "xs" as SizeKey, label: "XS", px: "12px" },
									{ key: "sm" as SizeKey, label: "S",  px: "14px" },
									{ key: "md" as SizeKey, label: "M",  px: "17px" },
									{ key: "lg" as SizeKey, label: "L",  px: "20px" },
								]}>
									{(opt) => (
										<button
											type="button"
											onClick={() => update("size", opt.key)}
											class={cn(
												"flex flex-1 flex-col items-center gap-0.5 rounded-md border-[1.5px] bg-transparent px-1 py-1.5 transition-[border-color,background]",
												t().size === opt.key
													? "border-primary bg-accent-subtle"
													: "border-border hover:border-border-strong",
											)}
										>
											<span
												class={cn(
													"block font-bold leading-none",
													t().size === opt.key ? "text-foreground" : "text-foreground",
												)}
												style={{ "font-size": opt.px }}
											>
												Aa
											</span>
											<span
												class={cn(
													"block text-[9.5px] font-semibold",
													t().size === opt.key ? "text-accent-text" : "text-faint",
												)}
											>
												{opt.label}
											</span>
										</button>
									)}
								</For>
							</div>
						</Section>

						{/* Sidebar width */}
						<Section label="Sidebar width">
							<SegControl
								options={[
									{ value: "narrow", label: "Narrow" },
									{ value: "default", label: "Default" },
									{ value: "wide", label: "Wide" },
								]}
								value={t().sidebarWidth}
								onChange={(v) => update("sidebarWidth", v as SidebarWidthKey)}
							/>
						</Section>

						{/* Row density */}
						<Section label="Row density">
							<SegControl
								options={[
									{ value: "compact", label: "Compact" },
									{ value: "default", label: "Default" },
									{ value: "spacious", label: "Spacious" },
								]}
								value={t().density}
								onChange={(v) => update("density", v as DensityKey)}
							/>
						</Section>

						{/* Corner radius */}
						<Section label="Corner radius">
							<SegControl
								options={[
									{ value: "sharp", label: "Sharp" },
									{ value: "default", label: "Default" },
									{ value: "round", label: "Round" },
								]}
								value={t().radius}
								onChange={(v) => update("radius", v as RadiusKey)}
							/>
						</Section>
					</div>

					{/* Footer */}
					<div class="flex shrink-0 items-center justify-between border-t border-border px-3.5 py-2.5">
						<button
							type="button"
							onClick={reset}
							class="text-[11.5px] text-faint transition-colors hover:text-foreground hover:underline"
						>
							Reset defaults
						</button>
						<span class="font-mono text-[10.5px] text-faint">
							{t().theme} · {FONTS[t().font].name} · {t().size.toUpperCase()}
						</span>
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
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
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

interface SectionProps {
	label: string;
	hint?: string;
	children: JSX.Element;
}

function Section(props: SectionProps) {
	return (
		<div class="border-t border-border px-3.5 py-2.5">
			<p class="mb-2 flex items-baseline gap-1.5 text-[10px] font-bold uppercase tracking-[0.08em] text-faint">
				{props.label}
				<Show when={props.hint}>
					<span class="text-[10px] font-normal normal-case tracking-normal opacity-65">{props.hint}</span>
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
