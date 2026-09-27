import { expect, test } from "../src/fixtures";

test.describe("Keyboard operability", () => {
	test("sorts the jobs table and opens a job with the keyboard", async ({
		page,
		jobsPage,
	}) => {
		await jobsPage.goto();

		const suitabilityHeader = page.getByRole("columnheader", {
			name: /suitability/i,
		});
		const sortButton = suitabilityHeader.getByRole("button");
		await expect(suitabilityHeader).toHaveAttribute("aria-sort", "none");

		await sortButton.focus();
		await page.keyboard.press("Enter");
		await expect(suitabilityHeader).toHaveAttribute("aria-sort", "descending");

		await page.keyboard.press("Enter");
		await expect(suitabilityHeader).toHaveAttribute("aria-sort", "ascending");

		const firstLink = jobsPage.firstJobTitleLink();
		const title = (await firstLink.textContent())?.trim() ?? "";
		await firstLink.focus();
		await page.keyboard.press("Enter");

		await expect(page).toHaveURL(/\/jobs\/.+/);
		await expect(page.getByRole("heading", { name: title })).toBeVisible();
	});

	test("sorts the CV table and opens a CV's PDF view with the keyboard", async ({
		page,
	}) => {
		await page.goto("/cv-templates");
		await expect(
			page.getByRole("heading", { name: "CVs", level: 1 }),
		).toBeVisible();

		const titleHeader = page.getByRole("columnheader", { name: /title/i });
		const sortButton = titleHeader.getByRole("button");
		await sortButton.focus();
		await page.keyboard.press("Enter");
		await expect(titleHeader).toHaveAttribute("aria-sort", "descending");

		const firstTitleLink = page
			.getByRole("row")
			.filter({ hasNot: page.getByRole("columnheader") })
			.first()
			.getByRole("link")
			.first();
		await firstTitleLink.focus();
		await page.keyboard.press("Enter");

		await expect(page).toHaveURL(/\/cv-templates\/.+\/.+/);
	});
});
