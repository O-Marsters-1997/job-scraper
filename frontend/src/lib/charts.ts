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
const FAINT = "#94a3b8";
const MUTED = "#475569";
const FG = "#0f172a";
const GRID = "#f1f5f9";

// Time-series line chart with zoom/pan — used by the Jobs discovered chart.
// Drag = box-zoom, scroll = zoom, shift+drag = pan.
export const lineChartOptions: ChartOptions<"line"> = {
	maintainAspectRatio: false,
	plugins: {
		legend: { display: false },
		tooltip: {
			backgroundColor: FG,
			padding: 8,
			titleFont: { family: MONO, size: 11 },
			bodyFont: { family: MONO, size: 11 },
		},
		zoom: {
			zoom: {
				wheel: { enabled: true },
				drag: {
					enabled: true,
					backgroundColor: "rgba(15,157,146,0.12)",
					borderColor: "#0f9d92",
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
				color: FAINT,
				font: { family: MONO, size: 10 },
				maxTicksLimit: 7,
			},
		},
		y: {
			grid: { color: GRID },
			border: { display: false },
			ticks: {
				color: FAINT,
				font: { family: MONO, size: 10 },
				precision: 0,
			},
			beginAtZero: true,
		},
	},
};

export const donutChartOptions: ChartOptions<"doughnut"> = {
	maintainAspectRatio: false,
	cutout: "70%",
	plugins: {
		legend: {
			display: true,
			position: "right",
			labels: {
				color: MUTED,
				font: { family: MONO, size: 11 },
				boxWidth: 10,
				padding: 12,
			},
		},
		tooltip: {
			backgroundColor: FG,
			padding: 8,
			titleFont: { family: MONO, size: 11 },
			bodyFont: { family: MONO, size: 11 },
		},
	},
};

// Legend is hidden — replaced by custom HTML chips in insights.tsx
export const stackedBarOptions: ChartOptions<"bar"> = {
	maintainAspectRatio: false,
	indexAxis: "y",
	plugins: {
		legend: { display: false },
		tooltip: {
			backgroundColor: FG,
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
				color: FAINT,
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

// Hex approximations for source OKLCH tokens — Canvas can't read CSS custom properties.
export const SOURCE_HEX: Record<string, string> = {
	LinkedIn: "#2867b2",
	Indeed: "#6011d1",
	Greenhouse: "#24a054",
	Lever: "#f57b3b",
};

export function sourceHex(source: string): string {
	return SOURCE_HEX[source] ?? "#64748b";
}

// Append a two-hex-digit alpha to a colour string, e.g. hexAlpha("#0f9d92", "1a").
export function hexAlpha(hex: string, alpha: string): string {
	return `${hex}${alpha}`;
}
