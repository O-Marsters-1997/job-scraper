import { expect, test } from "../src/fixtures";

test.describe("Insights", () => {
	test("switching theme preset recolours the charts without navigating", async ({
		page,
		tweaksPanel,
		insightsPage,
	}) => {
		await page.goto("/overview");
		await page.addStyleTag({
			content:
				'[aria-label="Open TanStack Router Devtools"] { display: none !important; }',
		});
		await insightsPage.gotoFromNav();

		const before = await insightsPage.snapshot();

		await tweaksPanel.open();
		await tweaksPanel.panel.getByRole("button", { name: "Midnight" }).click();

		await expect.poll(() => insightsPage.snapshot()).not.toBe(before);
		await expect(page).toHaveURL(/\/insights$/);
	});
});
