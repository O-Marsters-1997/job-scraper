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
		await expect(mobileNav.link("Overview")).toBeVisible();

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
		await mobileNav.link("Companies").click();

		await expect(page).toHaveURL(/\/companies$/);
		await expect(mobileNav.dialog).toBeHidden();
	});
});
