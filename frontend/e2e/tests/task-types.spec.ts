import { expect, test } from "../src/fixtures";

test.describe("Task types", () => {
	test("a duplicate name is rejected inline", async ({ taskTypesPage }) => {
		await taskTypesPage.goto();

		await taskTypesPage.add("Apply");

		await expect(
			taskTypesPage.main.getByText(
				"A task type with that name already exists.",
			),
		).toBeVisible();
	});

	test("renaming a type shows on the Tasks page", async ({
		taskTypesPage,
		tasksPage,
		page,
	}) => {
		await taskTypesPage.goto();
		await taskTypesPage.rename("Follow up", "Chase");
		await expect(taskTypesPage.row("Chase")).toBeVisible();

		await page
			.getByRole("link", { name: /^Tasks/ })
			.first()
			.click();
		await expect(tasksPage.rows.filter({ hasText: "Chase" })).toHaveCount(1);
	});

	test("deleting a type states the task count and cascades", async ({
		taskTypesPage,
		tasksPage,
		page,
	}) => {
		await taskTypesPage.goto();
		await taskTypesPage.startDelete("Reach out");
		await expect(taskTypesPage.dialog).toContainText("1 task will be deleted");

		await taskTypesPage.dialog.getByRole("button", { name: "Cancel" }).click();
		await expect(taskTypesPage.row("Reach out")).toBeVisible();

		await taskTypesPage.startDelete("Reach out");
		await taskTypesPage.dialog
			.getByRole("button", { name: "Delete task type" })
			.click();
		await expect(taskTypesPage.row("Reach out")).toHaveCount(0);

		await page
			.getByRole("link", { name: /^Tasks/ })
			.first()
			.click();
		await expect(tasksPage.rows).toHaveCount(2);
	});
});
