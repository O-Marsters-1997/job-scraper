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

// Pill metrics (CSS px, scaled per-frame for depth).
const ROLE_PX = 13;
const SRC_PX = 10;
const DOT = 15;
const PAD_X = 12;
const DOT_GAP = 9;
const LABEL_GAP = 7;
const PILL_H = 31;

// Depth → presentation. z in [0,1]: 0 far/small/faint/blurred, 1 near/large/sharp.
const SCALE_FAR = 0.62;
const SCALE_NEAR = 1.16;
const ALPHA_FAR = 0.55;
const ALPHA_NEAR = 1;
const BLUR_FAR = 2.8; // px of defocus on the farthest pills
const FOCUS_Z = 0.55; // z at/above which pills are perfectly sharp

// Motion (normalized field units per second). Constant per pill — DVD-style:
// straight-line travel, direction only ever flips on a wall bounce.
const SPD_MIN = 0.026;
const SPD_MAX = 0.07;
const VZ_MIN = 0.05;
const VZ_MAX = 0.13;

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

type Pill = {
	role: string;
	srcLabel: string;
	srcKey: SourceKey;
	// live state in normalized field space ([0,1] x/y, z depth) + constant velocity
	x: number;
	y: number;
	z: number;
	vx: number;
	vy: number;
	vz: number;
	// text metrics, re-measured each frame
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

		// One shared deterministic stream so every pill's coordinates are independent.
		// Stratified start: one pill per cell of a jittered grid, so x/y are always
		// evenly scattered (8 purely-random points clump together too often), while
		// depth and heading stay fully random. Each keeps its direction forever.
		const rng = mulberry32(0x9e3779b1);
		const COLS = 4;
		const ROWS = 2;
		const pills: Pill[] = JOBS.map(([role, key], idx) => {
			const col = idx % COLS;
			const row = Math.floor(idx / COLS);
			const cx = (col + 0.5 + (rng() - 0.5) * 0.8) / COLS;
			const cy = (row + 0.5 + (rng() - 0.5) * 0.8) / ROWS;
			const ang = rng() * TAU;
			const spd = SPD_MIN + rng() * (SPD_MAX - SPD_MIN);
			const vzMag = VZ_MIN + rng() * (VZ_MAX - VZ_MIN);
			return {
				role,
				srcLabel: SOURCES[key].label.toUpperCase(),
				srcKey: key,
				x: lerp(0.08, 0.92, cx),
				y: lerp(0.12, 0.88, cy),
				z: lerp(0.12, 0.9, rng()),
				vx: Math.cos(ang) * spd,
				vy: Math.sin(ang) * spd,
				vz: rng() < 0.5 ? -vzMag : vzMag,
				w: 0,
				roleW: 0,
			};
		});

		let cssW = 0;
		let cssH = 0;
		let dpr = 1;
		// False until the parent has a real layout box. The brand panel measures
		// 0×0 on mount (laid out a frame later), and integrating then would divide
		// by a ~1px width — the bounce margins balloon and clamp() piles every pill
		// at dead center. We hold motion until a genuine size arrives.
		let ready = false;
		const resize = () => {
			const parent = el.parentElement;
			const rect = parent
				? parent.getBoundingClientRect()
				: el.getBoundingClientRect();
			ready = rect.width > 1 && rect.height > 1;
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
			const blur = z >= FOCUS_Z ? 0 : lerp(BLUR_FAR, 0, z / FOCUS_Z);
			const w = p.w;
			const h = PILL_H;
			ctx.save();
			ctx.translate(x, y);
			// depth-of-field: far pills defocus so an overlapping nearer pill reads as in front
			if (blur > 0.05) ctx.filter = `blur(${blur}px)`;
			ctx.scale(s, s);
			ctx.globalAlpha = lerp(ALPHA_FAR, ALPHA_NEAR, z);

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
			ctx.beginPath();
			ctx.roundRect(cx, -DOT / 2, DOT, DOT, 5);
			ctx.fillStyle = `color-mix(in oklch, ${srcColor[p.srcKey]} 50%, transparent)`;
			ctx.fill();
			cx += DOT + DOT_GAP;

			ctx.textBaseline = "middle";
			ctx.textAlign = "left";
			ctx.font = roleFont;
			ctx.fillStyle = inkRole;
			ctx.letterSpacing = "0px";
			ctx.fillText(p.role, cx, 1);
			cx += p.roleW + LABEL_GAP;

			ctx.font = srcFont;
			ctx.fillStyle = srcColor[p.srcKey];
			ctx.letterSpacing = "0.05em";
			ctx.fillText(p.srcLabel, cx, 1);
			ctx.letterSpacing = "0px";

			ctx.restore();
		};

		const order = pills.map((_, i) => i);

		// Physics step: constant-velocity straight-line travel with wall bounces on
		// all three axes (x, y, depth). Direction is only ever flipped by a bounce —
		// speed and heading are otherwise preserved forever, so motion never repeats
		// or resets. A bounce negates one component, exactly like the DVD logo.
		const integrate = (dt: number) => {
			for (const p of pills) {
				p.x += p.vx * dt;
				p.y += p.vy * dt;
				p.z += p.vz * dt;

				const s = lerp(SCALE_FAR, SCALE_NEAR, p.z);
				const mx = Math.min(0.45, ((p.w * s) / 2 + 2) / cssW);
				const my = Math.min(0.45, ((PILL_H * s) / 2 + 2) / cssH);
				if (p.x < mx) {
					p.x = mx + (mx - p.x);
					p.vx = Math.abs(p.vx);
				} else if (p.x > 1 - mx) {
					p.x = 1 - mx - (p.x - (1 - mx));
					p.vx = -Math.abs(p.vx);
				}
				if (p.y < my) {
					p.y = my + (my - p.y);
					p.vy = Math.abs(p.vy);
				} else if (p.y > 1 - my) {
					p.y = 1 - my - (p.y - (1 - my));
					p.vy = -Math.abs(p.vy);
				}
				p.x = clamp(p.x, mx, 1 - mx);
				p.y = clamp(p.y, my, 1 - my);

				if (p.z < 0.04) {
					p.z = 0.08 - p.z;
					p.vz = Math.abs(p.vz);
				} else if (p.z > 0.96) {
					p.z = 1.92 - p.z;
					p.vz = -Math.abs(p.vz);
				}
			}
		};

		const paint = () => {
			ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
			ctx.clearRect(0, 0, cssW, cssH);
			measure();
			// painter's algorithm: far (small z) first, near last
			order.sort((a, b) => pills[a].z - pills[b].z);
			for (const i of order) {
				const p = pills[i];
				drawPill(p, p.x * cssW, p.y * cssH, p.z);
			}
		};

		resize();

		const ro =
			typeof ResizeObserver !== "undefined"
				? new ResizeObserver(() => {
						resize();
						// The animation loop repaints itself; the static frame doesn't, so
						// redraw it here once the parent finally reports a real size.
						if (reduce.matches && ready) paint();
					})
				: null;
		if (ro && el.parentElement) ro.observe(el.parentElement);

		const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");
		let raf = 0;
		let prev = 0;
		const loop = (now: number) => {
			if (!prev) prev = now;
			let dt = (now - prev) / 1000;
			prev = now;
			if (dt > 0.05) dt = 0.05; // cap big gaps (tab refocus) so nothing teleports
			// Hold the seeded (stratified) layout until layout is real; otherwise the
			// first frames would collapse every pill to the center.
			if (ready) {
				integrate(dt);
				paint();
			}
			raf = requestAnimationFrame(loop);
		};

		const startMotion = () => {
			cancelAnimationFrame(raf);
			if (reduce.matches) {
				paint(); // static frame at the seeded positions
			} else {
				prev = 0;
				raf = requestAnimationFrame(loop);
			}
		};
		startMotion();
		reduce.addEventListener?.("change", startMotion);
		// In the static (reduced-motion) case the rAF loop isn't re-measuring,
		// so repaint once when webfonts finish loading.
		document.fonts?.ready.then(() => {
			if (reduce.matches) paint();
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
