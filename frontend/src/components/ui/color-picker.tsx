import { ColorArea } from "@kobalte/core/color-area";
import { ColorSlider } from "@kobalte/core/color-slider";
import type { Color } from "@kobalte/core/colors";
import { parseColor } from "@kobalte/core/colors";
import { Popover } from "@kobalte/core/popover";
import { createSignal } from "solid-js";
import { cn } from "@/lib/utils";

interface ColorPickerProps {
	value: string;
	onChange: (css: string) => void;
	label: string;
}

function safeParseColor(v: string): Color {
	try {
		return parseColor(v);
	} catch {
		return parseColor("#000000");
	}
}

export function ColorPicker(props: ColorPickerProps) {
	const [color, setColor] = createSignal<Color>(
		safeParseColor(props.value).toFormat("hsba"),
	);

	function emitColor(c: Color) {
		const hsba = c.toFormat("hsba");
		setColor(hsba);
		const alpha = hsba.getChannelValue("alpha");
		const out =
			alpha < 1 ? hsba.toString("rgba") : hsba.toString("hex");
		props.onChange(out);
	}

	const thumbClass =
		"absolute size-3.5 -translate-x-1/2 -translate-y-1/2 cursor-grab rounded-full border-2 border-white shadow-md active:cursor-grabbing";

	return (
		<Popover>
			<Popover.Trigger
				type="button"
				class="size-6 shrink-0 rounded border border-border shadow-sm transition-transform hover:scale-110 focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
				style={{ background: props.value }}
				aria-label={`Edit ${props.label}`}
			/>
			<Popover.Portal>
				<Popover.Content
					data-tweaks-portal=""
					class={cn(
						"z-[60] flex w-52 flex-col gap-2.5 rounded-xl border border-border bg-surface p-3 shadow-xl outline-none",
						"data-[expanded]:animate-in data-[closed]:animate-out data-[closed]:fade-out-0 data-[expanded]:fade-in-0 data-[closed]:zoom-out-95 data-[expanded]:zoom-in-95",
					)}
				>
					{/* Saturation + brightness area */}
					<ColorArea
						value={color()}
						onChange={emitColor}
						xChannel="saturation"
						yChannel="brightness"
						colorSpace="hsb"
						class="relative h-32 w-full select-none overflow-hidden rounded-md"
					>
						<ColorArea.Background class="absolute inset-0">
							<ColorArea.Thumb class={cn(thumbClass, "top-0 left-0")} />
							<ColorArea.HiddenInputX />
							<ColorArea.HiddenInputY />
						</ColorArea.Background>
					</ColorArea>

					{/* Hue slider */}
					<ColorSlider
						value={color()}
						onChange={emitColor}
						channel="hue"
						colorSpace="hsb"
						class="w-full"
					>
						<ColorSlider.Track class="relative h-3 w-full overflow-hidden rounded-full">
							<ColorSlider.Thumb class={cn(thumbClass, "top-1/2")}>
								<ColorSlider.Input />
							</ColorSlider.Thumb>
						</ColorSlider.Track>
					</ColorSlider>

					{/* Alpha slider — checkers div underneath shows through transparent gradient */}
					<ColorSlider
						value={color()}
						onChange={emitColor}
						channel="alpha"
						colorSpace="hsb"
						class="w-full"
					>
						<div
							class="relative h-3 w-full overflow-hidden rounded-full"
							style={{
								background:
									"repeating-conic-gradient(#cbd5e1 0% 25%, white 0% 50%) 0 0 / 6px 6px",
							}}
						>
							<ColorSlider.Track class="absolute inset-0 h-full w-full rounded-full">
								<ColorSlider.Thumb class={cn(thumbClass, "top-1/2")}>
									<ColorSlider.Input />
								</ColorSlider.Thumb>
							</ColorSlider.Track>
						</div>
					</ColorSlider>

					{/* Hex input */}
					<div class="flex items-center gap-1.5">
						<div
							class="size-5 shrink-0 rounded border border-border"
							style={{ background: props.value }}
							aria-hidden="true"
						/>
						<input
							type="text"
							value={color().toString("hex")}
							onInput={(e) => {
								const raw = e.currentTarget.value.trim();
								try {
									const c = parseColor(raw);
									const alpha = color().getChannelValue("alpha");
									emitColor(
										c.toFormat("hsba").withChannelValue("alpha", alpha),
									);
								} catch {
									// Ignore partial / invalid input
								}
							}}
							class="h-6 w-full rounded-md border border-border bg-background px-2 font-mono text-[11px] text-foreground placeholder:text-faint focus:border-primary focus:outline-none"
							placeholder="#rrggbb"
							spellcheck={false}
							autocomplete="off"
						/>
					</div>
				</Popover.Content>
			</Popover.Portal>
		</Popover>
	);
}
