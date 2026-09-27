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

		const count = await companiesPage.rows.count();
		expect(count).toBeGreaterThan(0);
		for (let i = 0; i < count; i++) {
			const text = await companiesPage.rows.nth(i).textContent();
			expect(text?.toLowerCase()).toContain(firstName.trim().toLowerCase());
		}
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
		const toggle = companiesPage.rows.first().getByRole("group");
		await expect(toggle).toBeVisible();

		const wasChecked = await toggle.evaluate((el) =>
			el.hasAttribute("data-checked"),
		);
		await toggle.click();
		await expect
			.poll(() => toggle.evaluate((el) => el.hasAttribute("data-checked")))
			.toBe(!wasChecked);
	});
});
