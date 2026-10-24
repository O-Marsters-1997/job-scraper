import { expect, test } from "../src/fixtures";
import { SAVED_TIMEOUT_MS } from "../src/pages/draft.page";

const FIRST = "Senior Backend Engineer, Acme Ltd";
const SECOND = "Software Engineer, Globex";

test.describe("Roles drawer", () => {
	test.beforeEach(async ({ page, draftPage }) => {
		await draftPage.goto();
		await page.getByRole("button", { name: "Roles" }).click();
		await expect(draftPage.rolesDrawer).toBeVisible();
	});

	test("lists bank roles newest first with CV roles numbered", async ({
		draftPage,
	}) => {
		const rows = draftPage.rolesDrawer.getByRole("option");
		await expect(rows.nth(0)).toContainText(FIRST);
		await expect(rows.nth(0).getByRole("button")).toHaveText("1");
		await expect(rows.nth(1)).toContainText(SECOND);
		await expect(rows.nth(1).getByRole("button")).toHaveText("2");
	});

	test("1-9 moves a role's heading and bullets together and saves", async ({
		page,
		draftPage,
	}) => {
		const list = draftPage.rolesDrawer.getByRole("listbox");
		await draftPage.rolesDrawer.getByRole("option").nth(1).click();
		await list.press("1");
		const rows = draftPage.rolesDrawer.getByRole("option");
		await expect(rows.nth(1).getByRole("button")).toHaveText("1");
		await expect(rows.nth(0).getByRole("button")).toHaveText("2");
		const lines = page.getByRole("textbox", { name: /bullet/ });
		await expect(lines.first()).toHaveAccessibleName(/Software Engineer/);
		await expect(draftPage.saveStatus).toHaveText("Saved", {
			timeout: SAVED_TIMEOUT_MS,
		});
	});

	test("clicking a bank role adds its heading and best-fit bullet after the newer roles", async ({
		page,
		draftPage,
	}) => {
		const rows = draftPage.rolesDrawer.getByRole("option");
		await rows.nth(2).click();
		await expect(rows.nth(2).getByRole("button")).toHaveText("3");
		await expect(
			page
				.getByRole("main")
				.getByText("Developer, Initech")
				.locator("visible=true")
				.first(),
		).toBeVisible();
		await expect(
			page.getByRole("textbox", { name: /Developer bullet 1/ }),
		).toHaveCount(1);
		await expect(draftPage.saveStatus).toHaveText("Saved", {
			timeout: SAVED_TIMEOUT_MS,
		});
	});

	test("Choose its Achievements opens the selected role's drawer", async ({
		draftPage,
	}) => {
		const rows = draftPage.rolesDrawer.getByRole("option");
		await rows.nth(1).click();
		await draftPage.rolesDrawer
			.getByRole("button", { name: "Choose its Achievements" })
			.click();
		await expect(draftPage.drawer).toBeVisible();
		await expect(draftPage.drawer).toContainText("Software Engineer, Globex");
	});

	test("the circle removes a role, but never the last one", async ({
		page,
		draftPage,
	}) => {
		const rows = draftPage.rolesDrawer.getByRole("option");
		await rows.nth(1).getByRole("button").click();
		await expect(rows.nth(1).getByRole("button")).toHaveText("");
		await expect(page.getByRole("textbox", { name: /bullet/ })).toHaveCount(2);
		await rows.nth(0).getByRole("button").click();
		await expect(rows.nth(0).getByRole("button")).toHaveText("1");
	});
});
