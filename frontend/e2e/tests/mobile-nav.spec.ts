import { expect, test } from "../src/fixtures";

test.describe("Mobile navigation", () => {
	test.use({ viewport: { width: 375, height: 700 } });

	test("closed nav has no tab stops; opening traps focus; Escape restores it", async ({
		page,
		mobileNav,
	}) => {
		await page.goto("/jobs");
		await expect(page.getByRole("heading", { name: "Jobs" })).toBeVisible();

		await expect(mobileNav.toggle).toBeVisible();
		await expect(mobileNav.dialog).toBeHidden();
		await expect(
			page.getByRole("link", { name: "Overview", exact: true }),
		).toBeHidden();

		await mobileNav.open();
		await expect(mobileNav.link("Jobs")).toBeVisible();
		await expect(
			mobileNav.dialog.getByRole("link", { name: /^Applications/ }),
		).toBeVisible();
		for (const name of [
			"Overview",
			"Companies",
			"Experience",
			"Insights",
			"Settings",
		]) {
			await expect(mobileNav.link(name)).toHaveCount(0);
		}

		const focusInsideDialog = await page.evaluate(
			() => document.activeElement?.closest('[role="dialog"]') !== null,
		);
		expect(focusInsideDialog).toBe(true);

		await page.keyboard.press("Escape");
		await expect(mobileNav.dialog).toBeHidden();
		await expect(mobileNav.toggle).toBeFocused();
	});

	test("navigating to a link closes the dialog", async ({
		page,
		mobileNav,
	}) => {
		await page.goto("/jobs");
		await expect(page.getByRole("heading", { name: "Jobs" })).toBeVisible();

		await mobileNav.open();
		await mobileNav.dialog.getByRole("link", { name: /^Applications/ }).click();

		await expect(page).toHaveURL(/\/applications$/);
		await expect(mobileNav.dialog).toBeHidden();
	});
});
