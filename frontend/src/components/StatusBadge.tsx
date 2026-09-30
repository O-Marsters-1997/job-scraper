import { STATUS_FALLBACK_COLOUR } from "@/lib/status";

interface StatusBadgeProps {
	name: string;
	colour: string | undefined;
}

export function StatusBadge(props: StatusBadgeProps) {
	const colour = () => props.colour || STATUS_FALLBACK_COLOUR;
	return (
		<span
			class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium"
			style={{
				"background-color": `color-mix(in srgb, ${colour()} 14%, white)`,
				color: `color-mix(in srgb, ${colour()} 78%, black)`,
			}}
		>
			<span
				class="h-1.5 w-1.5 rounded-full"
				style={{ "background-color": colour() }}
			/>
			{props.name}
		</span>
	);
}
