import { expect, test } from "../src/fixtures";

test.describe("Tweaks panel", () => {
	test.beforeEach(async ({ page }) => {
		await page.goto("/overview");
		// The TanStack Router devtools launcher shares the bottom-right corner
		// in dev mode and can sit above the tweaks toggle in stacking order.
		await page.addStyleTag({
			content:
				'[aria-label="Open TanStack Router Devtools"] { display: none !important; }',
		});
	});

	test("editing a custom colour and clicking outside reverts the saved theme", async ({
		page,
		tweaksPanel,
	}) => {
		const savedPrimary = await tweaksPanel.primaryColor();

		await tweaksPanel.open();
		await tweaksPanel.openCustomEditor();
		await tweaksPanel.primaryColorTrigger().click();
		await tweaksPanel.hexInput.fill("#ff0000");

		await expect
			.poll(() => tweaksPanel.primaryColor())
			.not.toBe(savedPrimary);

		await page.mouse.click(10, 10);

		await expect.poll(() => tweaksPanel.primaryColor()).toBe(savedPrimary);
	});

	test("Escape closes the panel", async ({ tweaksPanel }) => {
		await tweaksPanel.open();
		await expect(tweaksPanel.toggle).toHaveAttribute("aria-expanded", "true");

		await tweaksPanel.toggle.page().keyboard.press("Escape");

		await expect(tweaksPanel.toggle).toHaveAttribute(
			"aria-expanded",
			"false",
		);
	});

	test("Sidebar width tweak resizes the sidebar", async ({
		appShell,
		tweaksPanel,
	}) => {
		const before = await appShell.root.boundingBox();

		await tweaksPanel.open();
		await tweaksPanel.panel.getByRole("button", { name: "Wide" }).click();

		await expect
			.poll(async () => (await appShell.root.boundingBox())?.width)
			.toBeGreaterThan(before?.width ?? 0);
	});
});
