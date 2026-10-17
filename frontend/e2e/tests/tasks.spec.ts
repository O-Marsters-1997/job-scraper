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
});
