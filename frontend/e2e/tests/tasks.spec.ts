import { expect, test } from "../src/fixtures";

test.describe("Tasks", () => {
	test("ticking a task removes it from the open list and the nav count", async ({
		tasksPage,
	}) => {
		await tasksPage.goto();
		await expect(tasksPage.rows).toHaveCount(3);
		await expect(tasksPage.navBadge).toContainText("3");

		await tasksPage.markFirstDone();

		await expect(tasksPage.rows).toHaveCount(2);
		await expect(tasksPage.navBadge).toContainText("2");
	});

	test("filter, sort and done survive a reload", async ({
		page,
		tasksPage,
	}) => {
		await tasksPage.goto();
		await tasksPage.filterByType("Apply");
		await tasksPage.sortByJobPosted();
		await expect(tasksPage.rows).toHaveCount(1);

		await tasksPage.markFirstDone();
		await expect(tasksPage.rows).toHaveCount(0);

		await tasksPage.toggleShowDone();
		await expect(tasksPage.rows).toHaveCount(1);

		await page.reload();
		await expect(page).toHaveURL(/type=/);
		await expect(page).toHaveURL(/sort=job_posted/);
		await expect(page).toHaveURL(/done=true/);
		await expect(
			page.getByRole("button", { name: "Show done" }),
		).toHaveAttribute("aria-pressed", "true");
	});
});
