import { expect, test } from "../src/fixtures";

test("should redirect to the first section and switch sections from the nav", async ({
	page,
}) => {
	await page.goto("/settings");
	await expect(page).toHaveURL(/\/settings\/profile$/);
	await expect(
		page.getByRole("heading", { name: "Profile", level: 1 }),
	).toBeVisible();

	const nav = page.getByRole("navigation", { name: "Settings sections" });
	await expect(nav.getByRole("link", { name: "Role" })).toBeHidden();

	await page.keyboard.press("]");
	await expect(nav.getByRole("link", { name: /^Scoring/ })).toBeFocused();
	await expect(page).toHaveURL(/\/settings\/profile$/);

	await page.keyboard.press("[");
	await expect(nav.getByRole("link", { name: /^Profile/ })).toBeFocused();
	await page.keyboard.press("]");
	await expect(nav.getByRole("link", { name: /^Scoring/ })).toBeFocused();

	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/settings\/scoring\/role$/);
	await expect(
		page.getByRole("heading", { name: "Scoring", level: 1 }),
	).toBeVisible();
	await expect(page.getByRole("button", { name: "Save" })).toBeVisible();

	await page.keyboard.press("]");
	await page.keyboard.press("]");
	await expect(
		nav.getByRole("link", { name: "Tech & industry" }),
	).toBeFocused();
	await page.keyboard.press("Enter");
	await expect(page).toHaveURL(/\/settings\/scoring\/stack$/);

	await page
		.getByRole("navigation", { name: "Breadcrumb" })
		.getByRole("link", { name: "Settings" })
		.click();
	await expect(page).toHaveURL(/\/settings\/profile$/);
});
