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

	test("grades one job from the row menu and keeps it on reopen", async ({
		page,
		jobsPage,
	}) => {
		const row = jobsPage.rows.first();
		await jobsPage.openRowMenu(row);
		await page.getByRole("menuitem", { name: "Grade…" }).click();
		const dialog = jobsPage.gradeDialog;
		await dialog.getByRole("button", { name: /^OK/ }).click();
		await dialog.getByRole("button", { name: "Salary" }).click();
		await dialog.getByRole("button", { name: "Save grade" }).click();
		await expect(dialog).toBeHidden();

		await expect(async () => {
			await jobsPage.openRowMenu(row);
			await page.getByRole("menuitem", { name: "Grade…" }).click({
				timeout: 2000,
			});
		}).toPass();
		await expect(dialog.getByRole("button", { name: /^OK/ })).toHaveAttribute(
			"aria-pressed",
			"true",
		);
		await expect(
			dialog.getByRole("button", { name: "Salary" }),
		).toHaveAttribute("aria-pressed", "true");
	});

	test("bulk grades the selected rows with one override and undoes it", async ({
		jobsPage,
	}) => {
		const dismissed = jobsPage.rows.nth(1).getByRole("link").first();
		const title = (await dismissed.textContent()) ?? "";
		const titleLink = jobsPage.page.getByRole("link", {
			name: title,
			exact: true,
		});
		await jobsPage.selectRows(3);
		await jobsPage.openBulkGrade();
		const dialog = jobsPage.gradeDialog;
		await dialog
			.getByRole("group", { name: "Grade for all" })
			.getByRole("button", { name: "OK" })
			.click();
		await dialog.getByRole("button", { name: "Next" }).click();
		await dialog.getByRole("button", { name: "No 3" }).click();
		await dialog.getByRole("button", { name: "Grade 3 jobs" }).click();
		await expect(dialog).toBeHidden();

		await expect(titleLink).toBeHidden();
		await jobsPage.undoBulkGrade();
		await expect(titleLink).toBeVisible();
	});

	test("shows a graded job's grade and drops it from the Ungraded filter", async ({
		page,
		jobsPage,
	}) => {
		const row = jobsPage.rows.first();
		const href =
			(await row.getByRole("link").first().getAttribute("href")) ?? "";
		const titleLink = page.locator(`a[href="${href}"]`);
		await jobsPage.openRowMenu(row);
		await page.getByRole("menuitem", { name: "Grade…" }).click();
		const dialog = jobsPage.gradeDialog;
		await dialog.getByRole("button", { name: /^OK/ }).click();
		await dialog.getByRole("button", { name: "Save grade" }).click();
		await expect(dialog).toBeHidden();
		await expect(
			page.getByRole("row").filter({ has: titleLink }).getByText("OK", {
				exact: true,
			}),
		).toBeVisible();

		await page.getByRole("button", { name: "Filters" }).click();
		await page.getByRole("button", { name: "Ungraded" }).click();
		await page.getByRole("button", { name: "Done" }).click();
		await expect(titleLink).toBeHidden();
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
