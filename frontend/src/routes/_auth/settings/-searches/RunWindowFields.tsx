import { For, Show } from "solid-js";
import { ToggleChip } from "@/components/ToggleChip";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectItemLabel,
	SelectTrigger,
} from "@/components/ui/select";
import {
	Switch,
	SwitchControl,
	SwitchLabel,
	SwitchThumb,
} from "@/components/ui/switch";
import {
	INTERVAL_OPTIONS,
	WEEKDAY_CHIPS,
	type WindowDraft,
} from "@/lib/runWindow";

type Option = { value: string; label: string };

export function RunWindowFields(props: {
	idPrefix: string;
	value: WindowDraft;
	onChange: (next: WindowDraft) => void;
	incremental: boolean;
	error: string | null;
}) {
	const automatic = () => props.value.automatic && props.incremental;
	const patch = (next: Partial<WindowDraft>) =>
		props.onChange({ ...props.value, ...next });
	const toggleDay = (day: number) =>
		patch({
			weekdays: props.value.weekdays.includes(day)
				? props.value.weekdays.filter((d) => d !== day)
				: [...props.value.weekdays, day],
		});

	return (
		<div class="flex flex-col gap-3">
			<div class="flex flex-col gap-1">
				<Switch
					checked={automatic()}
					disabled={!props.incremental}
					onChange={(automatic) => patch({ automatic })}
					class="flex items-center gap-2"
				>
					<SwitchControl>
						<SwitchThumb />
					</SwitchControl>
					<SwitchLabel class="text-xs font-medium text-foreground">
						Automatic
					</SwitchLabel>
				</Switch>
				<Show when={!props.incremental}>
					<p class="text-xs text-faint">
						This job board can't run automatically. Use Run now instead.
					</p>
				</Show>
			</div>

			<Show when={automatic()}>
				<div class="flex flex-wrap items-end gap-4">
					<Select<Option>
						class="flex w-44 flex-col gap-1.5"
						options={INTERVAL_OPTIONS}
						optionValue="value"
						optionTextValue="label"
						value={
							INTERVAL_OPTIONS.find(
								(o) => o.value === String(props.value.interval),
							) ?? null
						}
						onChange={(o) => o && patch({ interval: Number(o.value) })}
						itemComponent={(p) => (
							<SelectItem item={p.item}>
								<SelectItemLabel>{p.item.rawValue.label}</SelectItemLabel>
							</SelectItem>
						)}
					>
						<Select.Label class="text-xs font-medium text-foreground">
							Run
						</Select.Label>
						<SelectTrigger>
							<Select.Value<Option>>
								{(state) => state.selectedOption()?.label}
							</Select.Value>
						</SelectTrigger>
						<SelectContent />
					</Select>

					<div class="flex items-end gap-2">
						<div class="flex flex-col gap-1.5">
							<label
								for={`${props.idPrefix}-start`}
								class="text-xs font-medium text-foreground"
							>
								From
							</label>
							<Input
								id={`${props.idPrefix}-start`}
								type="time"
								value={props.value.start}
								onInput={(e) => patch({ start: e.currentTarget.value })}
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label
								for={`${props.idPrefix}-end`}
								class="text-xs font-medium text-foreground"
							>
								To
							</label>
							<Input
								id={`${props.idPrefix}-end`}
								type="time"
								value={props.value.end}
								onInput={(e) => patch({ end: e.currentTarget.value })}
							/>
						</div>
					</div>
				</div>

				<fieldset class="flex flex-wrap items-center gap-1">
					<legend class="sr-only">Days</legend>
					<For each={WEEKDAY_CHIPS}>
						{(d) => (
							<ToggleChip
								active={props.value.weekdays.includes(d.day)}
								onClick={() => toggleDay(d.day)}
								class="h-7 px-2.5"
							>
								{d.label}
							</ToggleChip>
						)}
					</For>
					<span class="ml-1 text-xs text-faint">{props.value.timezone}</span>
				</fieldset>
			</Show>

			<Show when={props.error}>
				<p role="alert" class="text-xs text-destructive-strong">
					{props.error}
				</p>
			</Show>
		</div>
	);
}
