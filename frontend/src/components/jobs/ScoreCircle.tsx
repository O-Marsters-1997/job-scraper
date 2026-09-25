import { scoreColour, tint } from "@/lib/scoreColour";

export function ScoreCircle(props: { value: number }) {
	const c = () => scoreColour(props.value);
	return (
		<span
			class="inline-flex size-7 items-center justify-center rounded-full text-xs font-semibold tabular-nums"
			style={tint(c(), { bg: 16 })}
		>
			{props.value}
		</span>
	);
}
