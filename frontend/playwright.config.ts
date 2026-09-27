import { defineConfig, devices } from "@playwright/test";

const isCI = !!process.env.CI;

export default defineConfig({
	testDir: "./e2e",
	outputDir: ".playwright/test-results",
	fullyParallel: true,
	forbidOnly: isCI,
	retries: isCI ? 2 : 0,
	...(isCI ? { workers: "50%" } : {}),
	reporter: isCI
		? [
				["html", { open: "never", outputFolder: ".playwright/report" }],
				["github"],
			]
		: [
				["html", { open: "on-failure", outputFolder: ".playwright/report" }],
				["list"],
			],
	use: {
		baseURL: "http://localhost:4444",
		trace: "on-first-retry",
		screenshot: "only-on-failure",
	},
	projects: [
		{
			name: "setup",
			testMatch: /auth\.setup\.ts/,
		},
		{
			name: "chromium",
			use: {
				...devices["Desktop Chrome"],
				storageState: ".playwright/.auth/user.json",
			},
			dependencies: ["setup"],
		},
	],
	webServer: {
		command: "bun run dev:test",
		url: "http://localhost:4444",
		env: { VITE_MOCK: "true" },
		reuseExistingServer: !isCI,
		timeout: 120_000,
	},
});
