import { BAND_COLOUR, BAND_LABEL } from "@/lib/band";
import { tintedChip } from "@/lib/scoreColour";
import type { Band } from "@/types/job";

export function BandChip(props: { band: Band }) {
	return (
		<span
			class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"
			style={tintedChip(BAND_COLOUR[props.band])}
		>
			{BAND_LABEL[props.band]}
		</span>
	);
}
