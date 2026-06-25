import { scoreColour } from "@/lib/scoreColour";

export function ScoreCircle(props: { value: number }) {
	const c = () => scoreColour(props.value);
	return (
		<span
			class="inline-flex size-7 items-center justify-center rounded-full text-xs font-semibold tabular-nums"
			style={{
				"background-color": `color-mix(in srgb, ${c()} 16%, white)`,
				color: `color-mix(in srgb, ${c()} 80%, black)`,
			}}
		>
			{props.value}
		</span>
	);
}
