import { For } from "solid-js";
import { ColorPicker } from "@/components/ui/color-picker";
import { oklchToHex, toOklch } from "@/lib/color";
import {
	type DensityKey,
	type FontKey,
	type RadiusKey,
	type SidebarWidthKey,
	type SizeKey,
	THEME_VAR_GROUPS,
} from "@/lib/tweaks";
import { FONT_KEYS, FONTS } from "@/lib/tweaks.themes";
import { cn } from "@/lib/utils";
import { Section, SegControl, selectableButtonVariants } from "./primitives";

export function TypographySection(props: {
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
							aria-pressed={props.selected === key}
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

export function SizeSection(props: {
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
							aria-pressed={props.selected === opt.key}
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

export function SidebarWidthSection(props: {
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

export function DensitySection(props: {
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

export function RadiusSection(props: {
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

export function CustomThemeEditor(props: {
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
