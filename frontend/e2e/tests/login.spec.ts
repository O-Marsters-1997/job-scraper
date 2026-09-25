import { expect, test } from "../src/fixtures";

test("should redirect to /jobs after successful login", async ({
	loginPage,
	page,
}) => {
	await loginPage.goto();
	await loginPage.login("demo", "demo");

	await expect(page).toHaveURL(/\/jobs/);
});

test("should render the sign-in form", async ({ loginPage, page }) => {
	await loginPage.goto();

	await expect(page.locator("#username")).toBeVisible();
	await expect(page.locator("#password")).toBeVisible();
	await expect(
		page.getByRole("button", { name: /sign in/i }),
	).toBeVisible();
});

test.fixme(
	"should show error on invalid credentials — needs real backend",
	async () => {},
);

test.fixme(
	"should redirect to /login when not authenticated — needs real backend",
	async () => {},
);
