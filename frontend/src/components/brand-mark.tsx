// FastTrack "Velocity F" monogram — an italic F whose arms trail two speed
// lines. Single-colour via currentColor so it adapts to the violet auth
// front door and the teal in-app sidebar alike.
export function FastTrackMark(props: { class?: string; size?: number }) {
	const size = () => props.size ?? 20;
	return (
		<svg
			aria-hidden="true"
			width={size()}
			height={size()}
			viewBox="0 0 32 32"
			fill="currentColor"
			class={props.class}
		>
			<g transform="skewX(-13) translate(3.4 0)">
				<path d="M9 6 L24 6 L24 10.5 L13.5 10.5 L13.5 14 L21 14 L21 18.5 L13.5 18.5 L13.5 26 L9 26 Z" />
			</g>
			<g stroke="currentColor" stroke-width="2.3" stroke-linecap="round">
				<path d="M2.5 9 H7" />
				<path d="M2 17 H6.5" />
			</g>
		</svg>
	);
}
