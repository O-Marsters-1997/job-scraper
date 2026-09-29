import { expect, test } from "../src/fixtures";

test.describe("Company boards tab", () => {
	test("pasting a Greenhouse URL tracks the company with an unverified board", async ({
		page,
	}) => {
		await page.goto("/settings/searches?tab=ats");
		await page
			.getByLabel("Company board URL")
			.fill("https://boards.greenhouse.io/zzunique");
		await page.getByRole("button", { name: /Track company/ }).click();

		const row = page.getByRole("row").filter({ hasText: "Zzunique" });
		await expect(row).toBeVisible();
		await expect(row.getByText("Unverified — not polling yet")).toBeVisible();

		await row.getByRole("button", { name: "Confirm board" }).click();
		await expect(row.getByText("Unverified — not polling yet")).toHaveCount(0);
	});

	test("untrack removes the row and Undo restores it", async ({ page }) => {
		await page.goto("/settings/searches?tab=ats");
		const first = page.getByRole("row").nth(1);
		const name = (await first.getByRole("link").first().innerText()).trim();

		await first.getByRole("button", { name: /^Untrack/ }).click();
		await expect(page.getByRole("link", { name, exact: true })).toHaveCount(0);
		await page.getByRole("button", { name: "Undo" }).click();
		await expect(page.getByRole("link", { name, exact: true })).toBeVisible();
	});

	test("companies page has no add form", async ({ page }) => {
		await page.goto("/companies");
		await expect(page.getByRole("button", { name: "Add company" })).toHaveCount(
			0,
		);
	});
});
