import { expect, test } from "../src/fixtures";

test.describe("Statuses", () => {
	test.beforeEach(async ({ statusesPage }) => {
		await statusesPage.goto();
	});

	test("renaming a status updates the list without a reload", async ({
		statusesPage,
	}) => {
		const newName = `Renamed Status ${Date.now()}`;

		await statusesPage.renameFirstStatus(newName);

		await expect(statusesPage.statusText(newName)).toBeVisible();
	});
});
