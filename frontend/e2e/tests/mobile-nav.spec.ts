import { expect, test } from "../src/fixtures";

test.describe("Mobile navigation", () => {
	test.use({ viewport: { width: 375, height: 700 } });

	test("closed nav has no tab stops; opening traps focus; Escape restores it", async ({
		page,
	}) => {
		await page.goto("/jobs");
		await expect(page.getByRole("heading", { name: "Jobs" })).toBeVisible();

		const toggle = page.getByRole("button", { name: "Open menu" });
		await expect(toggle).toBeVisible();

		await expect(page.getByRole("dialog")).toBeHidden();
		await expect(
			page.getByRole("link", { name: "Overview", exact: true }),
		).toBeHidden();

		await toggle.click();
		const dialog = page.getByRole("dialog");
		await expect(dialog).toBeVisible();
		await expect(dialog.getByRole("link", { name: "Overview" })).toBeVisible();

		const focusInsideDialog = await page.evaluate(
			() => document.activeElement?.closest('[role="dialog"]') !== null,
		);
		expect(focusInsideDialog).toBe(true);

		await page.keyboard.press("Escape");
		await expect(dialog).toBeHidden();
		await expect(toggle).toBeFocused();
	});
});
