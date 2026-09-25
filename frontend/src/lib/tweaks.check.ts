import { DEFAULTS, loadTweaks, STORAGE_KEY } from "./tweaks";

const saved = globalThis.localStorage;
globalThis.localStorage = {
	getItem: (key: string) =>
		key === STORAGE_KEY
			? JSON.stringify({
					font: "missing",
					radius: "missing",
					customColors: { "--color-primary": "red", "--other": "blue" },
				})
			: null,
} as Storage;
try {
	const tweaks = loadTweaks();
	if (tweaks.font !== DEFAULTS.font || tweaks.radius !== DEFAULTS.radius)
		throw new Error("unknown saved keys must use defaults");
	if (tweaks.customColors["--other"] !== undefined)
		throw new Error("unknown CSS variables must be ignored");
} finally {
	globalThis.localStorage = saved;
}
