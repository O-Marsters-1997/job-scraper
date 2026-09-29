import "chartjs-adapter-date-fns";
import {
	ArcElement,
	BarElement,
	CategoryScale,
	Chart,
	type ChartOptions,
	Filler,
	Legend,
	LinearScale,
	LineElement,
	PointElement,
	TimeScale,
	Title,
	Tooltip,
} from "chart.js";
import zoomPlugin from "chartjs-plugin-zoom";
import { createEffect, createMemo } from "solid-js";
import { cssVarHex } from "./color";
import { themeVersion } from "./tweaks";

const MONO = "'JetBrains Mono', monospace";

export function registerCharts() {
	Chart.register(
		Title,
		Tooltip,
		Legend,
		Filler,
		LineElement,
		PointElement,
		BarElement,
		ArcElement,
		CategoryScale,
		LinearScale,
		TimeScale,
		zoomPlugin,
	);

	Chart.defaults.maintainAspectRatio = false;
	Chart.defaults.plugins.legend.display = false;
	Chart.defaults.plugins.tooltip.padding = 8;
	Chart.defaults.plugins.tooltip.titleFont = { family: MONO, size: 11 };
	Chart.defaults.plugins.tooltip.bodyFont = { family: MONO, size: 11 };
	Chart.defaults.scale.ticks.font = { family: MONO, size: 10 };
	// Chart.defaults.scale's type is CoreChartOptions<"radar">["scale"], which has
	// no `border` — but Chart.js merges this object into every cartesian scale too.
	(Chart.defaults.scale as { border: { display: boolean } }).border = {
		display: false,
	};
}

// Canvas can't read CSS custom properties, so derive hex from live theme tokens
// at call time. This makes charts react to TweaksPanel preset switches.
export const primaryHex = () => cssVarHex("--color-primary");
export const sourceHex = (source: string) =>
	cssVarHex(
		`--color-source-${source.toLowerCase()}`,
		cssVarHex("--color-faint"),
	);

export function lineChartOptions(): ChartOptions<"line"> {
	const faint = cssVarHex("--color-faint");
	const fg = cssVarHex("--color-foreground");
	const grid = cssVarHex("--color-border");
	const primary = primaryHex();
	return {
		plugins: {
			tooltip: { backgroundColor: fg },
			zoom: {
				zoom: {
					wheel: { enabled: true },
					drag: {
						enabled: true,
						backgroundColor: `${primary}1e`,
						borderColor: primary,
						borderWidth: 1,
					},
					pinch: { enabled: true },
					mode: "x",
				},
				pan: {
					enabled: true,
					mode: "x",
					modifierKey: "shift",
				},
				limits: {
					x: { min: "original", max: "original" },
				},
			},
		},
		scales: {
			x: {
				type: "time",
				time: {
					unit: "day",
					tooltipFormat: "d MMM yyyy",
					displayFormats: { day: "d MMM" },
				},
				grid: { display: false },
				ticks: {
					color: faint,
					maxTicksLimit: 7,
				},
			},
			y: {
				grid: { color: grid },
				ticks: {
					color: faint,
					precision: 0,
				},
				beginAtZero: true,
			},
		},
	};
}

export function donutChartOptions(): ChartOptions<"doughnut"> {
	const muted = cssVarHex("--color-muted");
	const fg = cssVarHex("--color-foreground");
	return {
		cutout: "70%",
		plugins: {
			legend: {
				display: true,
				position: "right",
				labels: {
					color: muted,
					font: { family: MONO, size: 11 },
					boxWidth: 10,
					padding: 12,
				},
			},
			tooltip: { backgroundColor: fg },
		},
	};
}

export function stackedBarOptions(): ChartOptions<"bar"> {
	const faint = cssVarHex("--color-faint");
	const fg = cssVarHex("--color-foreground");
	return {
		indexAxis: "y",
		plugins: {
			tooltip: { backgroundColor: fg },
		},
		scales: {
			x: {
				stacked: true,
				grid: { display: false },
				ticks: {
					color: faint,
					precision: 0,
				},
			},
			y: {
				stacked: true,
				grid: { display: false },
				ticks: { display: false },
			},
		},
	};
}

export function horizontalBarOptions(): ChartOptions<"bar"> {
	const faint = cssVarHex("--color-faint");
	const muted = cssVarHex("--color-muted");
	const fg = cssVarHex("--color-foreground");
	const grid = cssVarHex("--color-border");
	return {
		indexAxis: "y",
		plugins: {
			tooltip: { backgroundColor: fg },
		},
		scales: {
			x: {
				grid: { color: grid },
				ticks: {
					color: faint,
					precision: 0,
				},
				beginAtZero: true,
			},
			y: {
				grid: { display: false },
				ticks: {
					color: muted,
				},
			},
		},
	};
}

export function verticalBarOptions(): ChartOptions<"bar"> {
	const faint = cssVarHex("--color-faint");
	const fg = cssVarHex("--color-foreground");
	const grid = cssVarHex("--color-border");
	return {
		plugins: {
			tooltip: { backgroundColor: fg },
		},
		scales: {
			x: {
				grid: { display: false },
				ticks: {
					color: faint,
				},
			},
			y: {
				grid: { color: grid },
				ticks: {
					color: faint,
					precision: 0,
				},
				beginAtZero: true,
			},
		},
	};
}

export const destructiveHex = () => cssVarHex("--color-destructive");

export { hexAlpha } from "./color";

// Canvas can't read CSS variables live, so chart colours are baked into
// plain values at build time; wrap any such builder in this so it recomputes
// when the applied theme changes.
export function createThemedMemo<T>(build: () => T): () => T {
	return createMemo(() => {
		themeVersion();
		return build();
	});
}

// Chart.js canvases carry their own role/aria-label since solid-chartjs
// doesn't forward arbitrary props to the underlying <canvas>.
export function useChartCanvasRef(
	label: () => string,
): (c: HTMLCanvasElement | null) => void {
	let canvas: HTMLCanvasElement | null = null;
	createEffect(() => {
		canvas?.setAttribute("aria-label", label());
	});
	return (c) => {
		canvas = c;
		c?.setAttribute("role", "img");
	};
}
