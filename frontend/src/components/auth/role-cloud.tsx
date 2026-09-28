import { For } from "solid-js";

type SourceKey = "li" | "gh" | "lv" | "in";

const SOURCE_LABEL: Record<SourceKey, string> = {
	li: "LinkedIn",
	gh: "Greenhouse",
	lv: "Lever",
	in: "Indeed",
};

type Pill = {
	role: string;
	src: SourceKey;
	x: number;
	y: number;
	z: number;
};

const FULL: Pill[] = [
	{ role: "Frontend Engineer", src: "li", x: 24, y: 16, z: 0.9 },
	{ role: "Product Designer", src: "gh", x: 72, y: 10, z: 0.35 },
	{ role: "Backend, Go", src: "lv", x: 50, y: 36, z: 0.7 },
	{ role: "ML Engineer", src: "in", x: 82, y: 42, z: 0.2 },
	{ role: "Platform Eng", src: "gh", x: 22, y: 58, z: 0.4 },
	{ role: "Growth Lead", src: "li", x: 62, y: 66, z: 1 },
	{ role: "Data Engineer", src: "in", x: 30, y: 88, z: 0.6 },
	{ role: "iOS Engineer", src: "lv", x: 78, y: 88, z: 0.5 },
];

const COMPACT: Pill[] = [
	{ role: "Frontend Engineer", src: "li", x: 30, y: 18, z: 0.85 },
	{ role: "Product Designer", src: "gh", x: 74, y: 28, z: 0.35 },
	{ role: "Backend, Go", src: "lv", x: 28, y: 62, z: 0.5 },
	{ role: "ML Engineer", src: "in", x: 70, y: 66, z: 0.95 },
];

export function RoleCloud(props: { variant?: "full" | "compact" }) {
	const compact = () => props.variant === "compact";
	return (
		<div class={compact() ? "role-cloud role-cloud-compact" : "role-cloud"}>
			<For each={compact() ? COMPACT : FULL}>
				{(pill, i) => (
					<span
						class="role-pill"
						style={{
							left: `${pill.x}%`,
							top: `${pill.y}%`,
							"--z": pill.z,
							"--src": `var(--auth-${pill.src})`,
							"--dx": `${i() % 2 ? -18 : 18}px`,
							"--dy": `${i() % 3 ? 12 : -12}px`,
							"z-index": Math.round(pill.z * 10),
							"animation-duration": `${11 + i() * 1.7}s`,
						}}
					>
						<span class="role-pill-dot" />
						{pill.role}
						<span class="role-pill-src">{SOURCE_LABEL[pill.src]}</span>
					</span>
				)}
			</For>
		</div>
	);
}
