import { expect, test } from "../src/fixtures";

test.describe("Scoring combobox", () => {
	test("supports arrow keys and exposes aria-expanded", async ({
		page,
		scoringPage,
	}) => {
		await scoringPage.gotoFilters();

		const combobox = scoringPage.excludedCompaniesCombobox;
		await expect(combobox).toHaveAttribute("aria-expanded", "false");

		await combobox.click();
		await expect(combobox).toHaveAttribute("aria-expanded", "true");

		await page.keyboard.press("ArrowDown");
		const activeDescendant = await combobox.getAttribute("aria-activedescendant");
		expect(activeDescendant).toBeTruthy();

		await page.keyboard.press("ArrowDown");
		const nextActiveDescendant = await combobox.getAttribute(
			"aria-activedescendant",
		);
		expect(nextActiveDescendant).toBeTruthy();
		expect(nextActiveDescendant).not.toBe(activeDescendant);

		await page.keyboard.press("Escape");
		await expect(combobox).toHaveAttribute("aria-expanded", "false");
	});
});
