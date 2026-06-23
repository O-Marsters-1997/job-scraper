import { onCleanup, onMount } from "solid-js";

// Canvas-rendered floating role cloud for the auth brand panel.
// Source-tagged role pills wander omnidirectionally with depth (forward/back),
// signifying the breadth of roles that stream in. Pure 2D canvas, no deps.

type SourceKey = "li" | "gh" | "lv" | "in";

const SOURCES: Record<
	SourceKey,
	{ label: string; varName: string; fallback: string }
> = {
	li: {
		label: "LinkedIn",
		varName: "--auth-li",
		fallback: "oklch(0.72 0.15 264)",
	},
	gh: {
		label: "Greenhouse",
		varName: "--auth-gh",
		fallback: "oklch(0.8 0.14 157)",
	},
	lv: { label: "Lever", varName: "--auth-lv", fallback: "oklch(0.78 0.15 40)" },
	in: {
		label: "Indeed",
		varName: "--auth-in",
		fallback: "oklch(0.74 0.16 300)",
	},
};

const JOBS: ReadonlyArray<readonly [string, SourceKey]> = [
	["Frontend Engineer", "li"],
	["Product Designer", "gh"],
	["Backend, Go", "lv"],
	["ML Engineer", "in"],
	["Platform Eng", "gh"],
	["Growth Lead", "li"],
	["Data Engineer", "in"],
	["iOS Engineer", "lv"],
];

// Scattered anchor points in normalized [0..1] field space, kept off the edges.
const ANCHORS: ReadonlyArray<readonly [number, number]> = [
	[0.18, 0.26],
	[0.62, 0.18],
	[0.43, 0.44],
	[0.8, 0.32],
	[0.16, 0.64],
	[0.56, 0.7],
	[0.33, 0.82],
	[0.74, 0.74],
];

// Pill metrics (CSS px, scaled per-frame for depth).
const ROLE_PX = 13;
const SRC_PX = 10;
const DOT = 15;
const PAD_X = 12;
const DOT_GAP = 9;
const LABEL_GAP = 7;
const PILL_H = 31;

// Depth → presentation. z in [0,1]: 0 far/small/faint, 1 near/large/bright.
const SCALE_FAR = 0.66;
const SCALE_NEAR = 1.14;
const ALPHA_FAR = 0.5;
const ALPHA_NEAR = 1;

const TAU = Math.PI * 2;
const lerp = (a: number, b: number, t: number) => a + (b - a) * t;
const clamp = (v: number, lo: number, hi: number) =>
	v < lo ? lo : v > hi ? hi : v;

// Deterministic PRNG (mulberry32) so layout/motion are stable across renders.
function mulberry32(seed: number) {
	let s = seed >>> 0;
	return () => {
		s = (s + 0x6d2b79f5) | 0;
		let t = Math.imul(s ^ (s >>> 15), 1 | s);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

// A wander oscillator: two incommensurate sine harmonics give smooth, organic,
// non-repeating motion (a Lissajous-style path) instead of a flat ping-pong.
type Osc = {
	a1: number;
	f1: number;
	p1: number;
	a2: number;
	f2: number;
	p2: number;
};
const makeOsc = (
	rng: () => number,
	ampBase: number,
	ampVar: number,
	fLo: number,
	fHi: number,
): Osc => {
	const f1 = fLo + rng() * (fHi - fLo);
	return {
		a1: ampBase + rng() * ampVar,
		f1,
		p1: rng() * TAU,
		a2: (ampBase + rng() * ampVar) * 0.55,
		f2: f1 * (1.7 + rng() * 0.6),
		p2: rng() * TAU,
	};
};
const evalOsc = (o: Osc, t: number) =>
	o.a1 * Math.sin(o.f1 * t + o.p1) + o.a2 * Math.sin(o.f2 * t + o.p2);

type Pill = {
	role: string;
	srcLabel: string;
	srcKey: SourceKey;
	ax: number;
	ay: number;
	ox: Osc;
	oy: Osc;
	oz: Osc;
	zBase: number;
	// metrics (filled once fonts are ready)
	w: number;
	roleW: number;
};

export function RoleCloud() {
	let canvas: HTMLCanvasElement | undefined;

	onMount(() => {
		const el = canvas;
		if (!el) return;
		const ctx = el.getContext("2d");
		if (!ctx) return;

		// Resolve themed colors from the element (inherits --auth-* from .auth-brand).
		const cs = getComputedStyle(el);
		const cssVar = (name: string, fallback: string) =>
			cs.getPropertyValue(name).trim() || fallback;
		const srcColor: Record<SourceKey, string> = {
			li: cssVar(SOURCES.li.varName, SOURCES.li.fallback),
			gh: cssVar(SOURCES.gh.varName, SOURCES.gh.fallback),
			lv: cssVar(SOURCES.lv.varName, SOURCES.lv.fallback),
			in: cssVar(SOURCES.in.varName, SOURCES.in.fallback),
		};
		const monoFamily = cssVar("--font-mono", "ui-monospace, monospace");
		const roleFont = `600 ${ROLE_PX}px "Plus Jakarta Sans", ui-sans-serif, sans-serif`;
		const srcFont = `600 ${SRC_PX}px ${monoFamily}`;
		const inkRole = "oklch(0.97 0.01 290)";
		const pillBg = "oklch(1 0 0 / 0.08)";
		const pillBorder = "oklch(1 0 0 / 0.16)";
		const glow = "oklch(0.7 0.15 290 / 0.55)";

		const pills: Pill[] = JOBS.map(([role, key], i) => {
			const rng = mulberry32((i + 1) * 0x9e3779b1);
			const [ax, ay] = ANCHORS[i % ANCHORS.length];
			return {
				role,
				srcLabel: SOURCES[key].label.toUpperCase(),
				srcKey: key,
				ax,
				ay,
				// Bigger travel than the CSS version, varied per pill, all directions.
				ox: makeOsc(rng, 0.07, 0.06, 0.1, 0.26),
				oy: makeOsc(rng, 0.07, 0.06, 0.1, 0.26),
				oz: makeOsc(rng, 0.32, 0.12, 0.07, 0.18),
				zBase: 0.5 + (rng() - 0.5) * 0.32,
				w: 0,
				roleW: 0,
			};
		});

		let cssW = 0;
		let cssH = 0;
		let dpr = 1;
		const resize = () => {
			const parent = el.parentElement;
			const rect = parent
				? parent.getBoundingClientRect()
				: el.getBoundingClientRect();
			cssW = Math.max(1, rect.width);
			cssH = Math.max(1, rect.height);
			dpr = Math.min(window.devicePixelRatio || 1, 2);
			el.width = Math.round(cssW * dpr);
			el.height = Math.round(cssH * dpr);
		};

		// Re-measured every frame with the exact font state used to draw, so pill
		// widths always match the rendered glyphs regardless of webfont load timing.
		const measure = () => {
			ctx.letterSpacing = "0px";
			for (const p of pills) {
				ctx.font = roleFont;
				p.roleW = ctx.measureText(p.role).width;
				ctx.font = srcFont;
				ctx.letterSpacing = "0.05em";
				const sw = ctx.measureText(p.srcLabel).width;
				ctx.letterSpacing = "0px";
				p.w = PAD_X * 2 + DOT + DOT_GAP + p.roleW + LABEL_GAP + sw;
			}
		};

		const drawPill = (p: Pill, x: number, y: number, z: number) => {
			const s = lerp(SCALE_FAR, SCALE_NEAR, z);
			const w = p.w;
			const h = PILL_H;
			ctx.save();
			ctx.translate(x, y);
			ctx.scale(s, s);
			ctx.globalAlpha = lerp(ALPHA_FAR, ALPHA_NEAR, z);

			// pill body
			ctx.beginPath();
			ctx.roundRect(-w / 2, -h / 2, w, h, h / 2);
			if (z > 0.62) {
				ctx.shadowColor = glow;
				ctx.shadowBlur = lerp(0, 26, (z - 0.62) / 0.38);
				ctx.shadowOffsetY = 5;
			}
			ctx.fillStyle = pillBg;
			ctx.fill();
			ctx.shadowColor = "transparent";
			ctx.shadowBlur = 0;
			ctx.shadowOffsetY = 0;
			ctx.lineWidth = 1;
			ctx.strokeStyle = pillBorder;
			ctx.stroke();

			let cx = -w / 2 + PAD_X;
			// source dot (rounded square)
			ctx.beginPath();
			ctx.roundRect(cx, -DOT / 2, DOT, DOT, 5);
			ctx.fillStyle = `color-mix(in oklch, ${srcColor[p.srcKey]} 50%, transparent)`;
			ctx.fill();
			cx += DOT + DOT_GAP;

			// role label
			ctx.textBaseline = "middle";
			ctx.textAlign = "left";
			ctx.font = roleFont;
			ctx.fillStyle = inkRole;
			ctx.letterSpacing = "0px";
			ctx.fillText(p.role, cx, 1);
			cx += p.roleW + LABEL_GAP;

			// source label
			ctx.font = srcFont;
			ctx.fillStyle = srcColor[p.srcKey];
			ctx.letterSpacing = "0.05em";
			ctx.fillText(p.srcLabel, cx, 1);
			ctx.letterSpacing = "0px";

			ctx.restore();
		};

		const order = pills.map((_, i) => i);
		const render = (t: number) => {
			ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
			ctx.clearRect(0, 0, cssW, cssH);
			measure();

			const z = new Float32Array(pills.length);
			for (let i = 0; i < pills.length; i++) {
				z[i] = clamp(pills[i].zBase + evalOsc(pills[i].oz, t), 0, 1);
			}
			// painter's algorithm: far (small z) first, near last
			order.sort((a, b) => z[a] - z[b]);

			for (const i of order) {
				const p = pills[i];
				const s = lerp(SCALE_FAR, SCALE_NEAR, z[i]);
				const halfW = (p.w * s) / 2 + 2;
				const halfH = (PILL_H * s) / 2 + 2;
				const x = clamp(
					p.ax * cssW + evalOsc(p.ox, t) * cssW,
					halfW,
					cssW - halfW,
				);
				const y = clamp(
					p.ay * cssH + evalOsc(p.oy, t) * cssH,
					halfH,
					cssH - halfH,
				);
				drawPill(p, x, y, z[i]);
			}
		};

		resize();

		const ro =
			typeof ResizeObserver !== "undefined" ? new ResizeObserver(resize) : null;
		if (ro && el.parentElement) ro.observe(el.parentElement);

		const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");
		let raf = 0;
		let t0 = 0;
		const loop = (now: number) => {
			if (!t0) t0 = now;
			render((now - t0) / 1000);
			raf = requestAnimationFrame(loop);
		};

		const startMotion = () => {
			cancelAnimationFrame(raf);
			if (reduce.matches) {
				t0 = 0;
				render(0); // static, scattered frame
			} else {
				raf = requestAnimationFrame(loop);
			}
		};
		startMotion();
		reduce.addEventListener?.("change", startMotion);
		// In the static (reduced-motion) case the rAF loop isn't re-measuring,
		// so repaint once when webfonts finish loading.
		document.fonts?.ready.then(() => {
			if (reduce.matches) render(0);
		});

		onCleanup(() => {
			cancelAnimationFrame(raf);
			ro?.disconnect();
			reduce.removeEventListener?.("change", startMotion);
		});
	});

	// Decorative; hidden from assistive tech via the aria-hidden .auth-field parent.
	return <canvas ref={canvas} class="auth-canvas" />;
}
