import { For } from "solid-js";
import { STATUS_PALETTE } from "@/lib/status";

export function StatusColourPicker(props: {
	value: string;
	onChange: (hex: string) => void;
}) {
	return (
		<div class="flex gap-1">
			<For each={STATUS_PALETTE}>
				{(p) => (
					<button
						type="button"
						title={p.label}
						onClick={() => props.onChange(p.hex)}
						class="h-5 w-5 rounded-full border-2 transition"
						style={{
							background: p.hex,
							"border-color":
								props.value === p.hex
									? "var(--color-foreground)"
									: "transparent",
						}}
					/>
				)}
			</For>
		</div>
	);
}
