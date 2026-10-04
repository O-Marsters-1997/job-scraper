import { expect, test } from "../src/fixtures";

test.describe("Companies", () => {
	test.beforeEach(async ({ companiesPage }) => {
		await companiesPage.goto();
	});

	test("should display company listings", async ({ companiesPage }) => {
		await expect(companiesPage.rows.first()).toBeVisible();
	});

	test("should filter companies by search term", async ({ companiesPage }) => {
		const firstName = await companiesPage.firstCompanyNameLink().textContent();
		if (!firstName) throw new Error("Could not read first company name");

		await companiesPage.search(firstName.trim());

		const needle = firstName.trim().toLowerCase();
		await expect
			.poll(async () => {
				const texts = await companiesPage.rows.allTextContents();
				return (
					texts.length > 0 &&
					texts.every((t) => t.toLowerCase().includes(needle))
				);
			})
			.toBe(true);
	});

	test("should navigate to company detail when clicking a company name", async ({
		page,
		companiesPage,
	}) => {
		const nameText = await companiesPage.firstCompanyNameLink().textContent();
		if (!nameText) throw new Error("Could not read first company name");
		await companiesPage.openFirstCompany();

		await expect(page).toHaveURL(/\/companies\/.+/);
		await expect(
			page.getByRole("heading", { name: nameText.trim(), level: 1 }),
		).toBeVisible();
	});

	test("should toggle tracking for a company with a known ATS board", async ({
		companiesPage,
	}) => {
		const name = await companiesPage.firstCompanyNameLink().textContent();
		if (!name) throw new Error("Could not read first company name");
		const toggle = companiesPage.rows
			.filter({ hasText: name.trim() })
			.first()
			.getByRole("group");
		await expect(toggle).toBeVisible();

		const wasChecked = await toggle.evaluate((el) =>
			el.hasAttribute("data-checked"),
		);
		await toggle.click();
		await expect
			.poll(() => toggle.evaluate((el) => el.hasAttribute("data-checked")))
			.toBe(!wasChecked);
	});

	test("a failing company query shows an error with retry, not 'Company not found'", async ({
		page,
		companiesPage,
	}) => {
		const name = (
			await companiesPage.firstCompanyNameLink().textContent()
		)?.trim();
		if (!name) throw new Error("Could not read first company name");
		await companiesPage.openFirstCompany();
		await expect(page).toHaveURL(/\/companies\/.+/);
		await page.evaluate(() =>
			localStorage.setItem("mock-fail", "getCompanies"),
		);
		await page.reload();

		await expect(page.getByText("Something went wrong")).toBeVisible({
			timeout: 10_000,
		});
		await expect(page.getByText("Company not found")).toBeHidden();

		await page.evaluate(() => localStorage.removeItem("mock-fail"));
		await page.getByRole("button", { name: "Try again" }).click();
		await expect(page.getByRole("heading", { name, level: 1 })).toBeVisible();
		await expect(page.getByText("Something went wrong")).toBeHidden();
	});

	test("Track application on the company page creates an application", async ({
		page,
		companiesPage,
	}) => {
		const withJobs = companiesPage.rows.filter({
			has: page.getByRole("cell", { name: /^[1-9]\d*$/ }),
		});
		await withJobs.first().getByRole("link").first().click();
		await expect(page).toHaveURL(/\/companies\/.+/);

		const jobRow = page
			.getByRole("row")
			.filter({ hasNot: page.getByRole("columnheader") })
			.first();
		await expect(jobRow).toBeVisible();
		await jobRow.getByRole("button", { name: "Job actions" }).click();
		await page.getByRole("menuitem", { name: "Track application" }).click();
		const dialog = page.getByRole("dialog");
		await dialog.getByLabel("Status").selectOption({ label: "Applied" });
		await dialog.getByRole("button", { name: "Save" }).click();

		await expect(dialog).toBeHidden();
		await expect(jobRow.getByText("Applied")).toBeVisible();
	});

	test.describe("at 375px", () => {
		test.use({ viewport: { width: 375, height: 700 } });

		test("the company detail page does not scroll horizontally", async ({
			page,
			companiesPage,
		}) => {
			await companiesPage.openFirstCompany();
			await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
			const overflow = await page.evaluate(
				() =>
					document.documentElement.scrollWidth -
					document.documentElement.clientWidth,
			);
			expect(overflow).toBeLessThanOrEqual(0);
		});
	});
});
