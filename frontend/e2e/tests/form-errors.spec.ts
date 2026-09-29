import { expect, test } from "../src/fixtures";

test.describe("Form errors", () => {
	test("a failing status create shows an inline error and logs no unhandled rejection", async ({
		page,
		statusesPage,
	}) => {
		const unhandled: string[] = [];
		page.on("pageerror", (err) => unhandled.push(err.message));
		await statusesPage.goto();
		await page.evaluate(() =>
			localStorage.setItem("mock-fail", "createStatus"),
		);

		await page.getByRole("button", { name: "Add status" }).click();
		await page.getByLabel("Status name").fill("Doomed");
		await page.getByLabel("Status name").press("Enter");

		await expect(
			page.getByText("Something went wrong. Please try again."),
		).toBeVisible();
		await expect(page.getByLabel("Status name")).toHaveValue("Doomed");
		expect(unhandled).toEqual([]);
	});

	test("pressing Enter twice quickly creates one status", async ({
		page,
		statusesPage,
	}) => {
		await statusesPage.goto();
		const name = `Twice ${Date.now()}`;

		await page.getByRole("button", { name: "Add status" }).click();
		const input = page.getByLabel("Status name");
		await input.fill(name);
		await input.press("Enter");
		await input.press("Enter");

		await expect(statusesPage.statusText(name)).toHaveCount(1);
	});

	test("a failing company add keeps the dialog open with an error", async ({
		page,
		companiesPage,
	}) => {
		await companiesPage.goto();
		await page.evaluate(() => localStorage.setItem("mock-fail", "addCompany"));

		await page.getByRole("button", { name: "Add company" }).click();
		const dialog = page.getByRole("dialog");
		await dialog
			.getByLabel("ATS board URL")
			.fill("https://boards.greenhouse.io/acmecorp");
		await dialog.getByRole("button", { name: "Add", exact: true }).click();

		await expect(
			dialog.getByText("Failed to add company. Please try again."),
		).toBeVisible();
		await expect(dialog).toBeVisible();
	});
});
