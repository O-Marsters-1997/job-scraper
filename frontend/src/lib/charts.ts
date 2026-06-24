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
import { cssVarHex } from "./color";

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

// Time-series line chart with zoom/pan — used by the Jobs discovered chart.
// Drag = box-zoom, scroll = zoom, shift+drag = pan.
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

// Legend is hidden — replaced by custom HTML chips in insights.tsx
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

// Append a two-hex-digit alpha to a colour string, e.g. hexAlpha("#6645d9", "1a").
export function hexAlpha(hex: string, alpha: string): string {
	return `${hex}${alpha}`;
}
