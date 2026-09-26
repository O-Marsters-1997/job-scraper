import { expect, test } from "../src/fixtures";

test("should open a section from the hub and return via the breadcrumb", async ({
	page,
}) => {
	await page.goto("/settings");
	await expect(
		page.getByRole("heading", { name: "Settings", level: 1 }),
	).toBeVisible();

	await page.getByRole("link", { name: "Scoring" }).click();
	await expect(page).toHaveURL(/\/settings\/scoring$/);
	await expect(
		page.getByRole("heading", { name: "Scoring settings" }),
	).toBeVisible();

	await page.getByRole("navigation", { name: "Breadcrumb" }).getByRole("link", { name: "Settings" }).click();
	await expect(page).toHaveURL(/\/settings$/);
});
