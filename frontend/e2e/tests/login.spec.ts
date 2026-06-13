import { expect, test } from "../src/fixtures";

// Deferred to real-backend phase (needs VITE_MOCK disabled):
//   - shows error on invalid credentials
//   - auth gate redirects unauthenticated users to /login

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
	async () => {
		// Switch VITE_MOCK off and use a real seeded test user to enable.
	},
);

test.fixme(
	"should redirect to /login when not authenticated — needs real backend",
	async () => {
		// Switch VITE_MOCK off; the _auth.tsx gate will redirect on getMe failure.
	},
);
