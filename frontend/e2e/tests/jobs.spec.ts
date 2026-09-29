import { expect, test } from "../src/fixtures";

test.describe("Jobs", () => {
	test.beforeEach(async ({ jobsPage }) => {
		await jobsPage.goto();
	});

	test("should display job listings", async ({ jobsPage }) => {
		await expect(jobsPage.rows.first()).toBeVisible();
	});

	test("should filter jobs by search term", async ({ jobsPage }) => {
		const firstTitle = await jobsPage.firstJobTitleLink().textContent();

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
		if (!titleText) throw new Error("Could not read first job title");
		await jobsPage.openFirstJob();

		await expect(page).toHaveURL(/\/jobs\/.+/);
		await expect(
			page.getByRole("heading", { name: titleText.trim() }),
		).toBeVisible();
	});

	test("tracking a job from the table updates its Status cell without a page or sort change", async ({
		jobsPage,
	}) => {
		await jobsPage.sortByCompany();
		const companyBeforeTracking = await jobsPage
			.firstRowCompanyText()
			.textContent();

		await jobsPage.trackFirstJob("Applied");

		await expect(jobsPage.rows.first().getByText("Applied")).toBeVisible();
		await expect(jobsPage.firstRowCompanyText()).toHaveText(
			companyBeforeTracking ?? "",
		);
	});

	test("shows the job title in the breadcrumb on a direct visit to /jobs/:id", async ({
		page,
		jobsPage,
	}) => {
		const link = jobsPage.firstJobTitleLink();
		const titleText = await link.textContent();
		if (!titleText) throw new Error("Could not read first job title");
		const href = await link.getAttribute("href");
		if (!href) throw new Error("Could not read job link href");

		await page.goto(href);

		await expect(
			page
				.getByRole("navigation", { name: "Breadcrumb" })
				.getByText(titleText.trim()),
		).toBeVisible();
	});

	test.describe("at 375px", () => {
		test.use({ viewport: { width: 375, height: 700 } });

		test("the job detail page does not scroll horizontally", async ({
			page,
			jobsPage,
		}) => {
			await jobsPage.openFirstJob();
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
