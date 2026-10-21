import { cn } from "@/lib/utils";

const F =
	"M9 6 L24 6 L24 10.5 L13.5 10.5 L13.5 14 L21 14 L21 18.5 L13.5 18.5 L13.5 26 L9 26 Z";

export function Spinner(props: { class?: string; decorative?: boolean }) {
	return (
		<svg
			viewBox="0 0 32 32"
			class={cn(
				"spinner size-5 shrink-0 overflow-visible text-primary",
				props.class,
			)}
			role={props.decorative ? undefined : "img"}
			aria-label={props.decorative ? undefined : "Loading"}
			aria-hidden={props.decorative ? "true" : undefined}
		>
			<rect
				class="spinner-hair spinner-frame-base"
				x="3"
				y="3"
				width="26"
				height="26"
				rx="6.5"
				pathLength="100"
			/>
			<rect
				class="spinner-hair spinner-frame-run"
				x="3"
				y="3"
				width="26"
				height="26"
				rx="6.5"
				pathLength="100"
			/>
			<g transform="translate(16 16) scale(0.8) translate(-15 -16)">
				<g transform="skewX(-13) translate(3.4 0)">
					<path
						class="spinner-outline spinner-comet-base"
						d={F}
						pathLength="100"
					/>
					<path class="spinner-outline spinner-comet" d={F} pathLength="100" />
				</g>
				<path class="spinner-line spinner-flick-1" d="M7 9 H2.5" />
				<path class="spinner-line spinner-flick-2" d="M6.5 17 H2" />
			</g>
		</svg>
	);
}

export function FullPageSpinner() {
	return (
		<div class="grid h-screen place-items-center bg-background">
			<Spinner class="size-28" />
		</div>
	);
}
