import assert from "node:assert/strict";
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
	assert.equal(tweaks.font, DEFAULTS.font);
	assert.equal(tweaks.radius, DEFAULTS.radius);
	assert.equal(tweaks.customColors["--other"], undefined);
} finally {
	globalThis.localStorage = saved;
}
