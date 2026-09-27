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
}

const MONO = "'JetBrains Mono', monospace";

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
		maintainAspectRatio: false,
		plugins: {
			legend: { display: false },
			tooltip: {
				backgroundColor: fg,
				padding: 8,
				titleFont: { family: MONO, size: 11 },
				bodyFont: { family: MONO, size: 11 },
			},
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
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
					maxTicksLimit: 7,
				},
			},
			y: {
				grid: { color: grid },
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
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
		maintainAspectRatio: false,
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
			tooltip: {
				backgroundColor: fg,
				padding: 8,
				titleFont: { family: MONO, size: 11 },
				bodyFont: { family: MONO, size: 11 },
			},
		},
	};
}

export function stackedBarOptions(): ChartOptions<"bar"> {
	const faint = cssVarHex("--color-faint");
	const fg = cssVarHex("--color-foreground");
	return {
		maintainAspectRatio: false,
		indexAxis: "y",
		plugins: {
			legend: { display: false },
			tooltip: {
				backgroundColor: fg,
				padding: 8,
				titleFont: { family: MONO, size: 11 },
				bodyFont: { family: MONO, size: 11 },
			},
		},
		scales: {
			x: {
				stacked: true,
				grid: { display: false },
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
					precision: 0,
				},
			},
			y: {
				stacked: true,
				grid: { display: false },
				border: { display: false },
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
		maintainAspectRatio: false,
		indexAxis: "y",
		plugins: {
			legend: { display: false },
			tooltip: {
				backgroundColor: fg,
				padding: 8,
				titleFont: { family: MONO, size: 11 },
				bodyFont: { family: MONO, size: 11 },
			},
		},
		scales: {
			x: {
				grid: { color: grid },
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
					precision: 0,
				},
				beginAtZero: true,
			},
			y: {
				grid: { display: false },
				border: { display: false },
				ticks: {
					color: muted,
					font: { family: MONO, size: 10 },
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
		maintainAspectRatio: false,
		plugins: {
			legend: { display: false },
			tooltip: {
				backgroundColor: fg,
				padding: 8,
				titleFont: { family: MONO, size: 11 },
				bodyFont: { family: MONO, size: 11 },
			},
		},
		scales: {
			x: {
				grid: { display: false },
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
				},
			},
			y: {
				grid: { color: grid },
				border: { display: false },
				ticks: {
					color: faint,
					font: { family: MONO, size: 10 },
					precision: 0,
				},
				beginAtZero: true,
			},
		},
	};
}

export const destructiveHex = () => cssVarHex("--color-destructive");

export function hexAlpha(hex: string, alpha: string): string {
	return `${hex}${alpha}`;
}

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
