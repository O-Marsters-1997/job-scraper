interface StatusBadgeProps {
	name: string;
	colour: string;
}

// Status pill matching the design's "colored dot + soft tinted background" rule.
export function StatusBadge(props: StatusBadgeProps) {
	return (
		<span
			class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-0.5 text-xs font-medium"
			style={{
				"background-color": `color-mix(in srgb, ${props.colour} 14%, white)`,
				color: `color-mix(in srgb, ${props.colour} 78%, black)`,
			}}
		>
			<span
				class="h-1.5 w-1.5 rounded-full"
				style={{ "background-color": props.colour }}
			/>
			{props.name}
		</span>
	);
}
