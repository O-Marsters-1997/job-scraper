import { expect, test } from "../src/fixtures";

test.describe("Jobs", () => {
	test.beforeEach(async ({ jobsPage }) => {
		await jobsPage.goto();
	});

	test("should display job listings", async ({ jobsPage }) => {
		await expect(jobsPage.rows.first()).toBeVisible();
	});

	test("should filter jobs by search term", async ({ jobsPage }) => {
		const firstTitle = await jobsPage
			.firstJobTitleLink()
			.textContent();

		if (!firstTitle) throw new Error("Could not read first job title");

		const term = firstTitle.trim().split(/\s+/)[0] as string;
		await jobsPage.search(term);

		const count = await jobsPage.rows.count();
		expect(count).toBeGreaterThan(0);
		for (let i = 0; i < count; i++) {
			const text = await jobsPage.rows.nth(i).textContent();
			expect(text?.toLowerCase()).toContain(term.toLowerCase());
		}
	});

	test("should navigate to job detail when clicking a job title", async ({
		page,
		jobsPage,
	}) => {
		const titleText = await jobsPage.firstJobTitleLink().textContent();
		await jobsPage.openFirstJob();

		await expect(page).toHaveURL(/\/jobs\/.+/);
		await expect(
			page.getByRole("heading", { name: titleText?.trim() }),
		).toBeVisible();
	});
});
