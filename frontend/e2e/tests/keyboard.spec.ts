import { expect, test } from "../src/fixtures";

test.describe("Keyboard operability", () => {
	test("tabs through the Draft's lines in page order and blurs on Escape", async ({
		page,
		draftPage,
	}) => {
		await draftPage.goto();

		await draftPage.line("Profile").focus();
		const order = [
			"Senior Backend Engineer bullet 1",
			"Senior Backend Engineer bullet 2",
			"Software Engineer bullet 1",
		];
		for (const label of order) {
			await page.keyboard.press("Tab");
			await expect(draftPage.line(label)).toBeFocused();
		}

		await page.keyboard.press("Enter");
		const text = await draftPage
			.line("Software Engineer bullet 1")
			.evaluate((el) => el.textContent);
		expect(text).not.toContain("\n");
		await page.keyboard.press("Escape");
		await expect(
			draftPage.line("Software Engineer bullet 1"),
		).not.toBeFocused();
	});

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
