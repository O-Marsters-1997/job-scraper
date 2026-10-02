import { devices } from "@playwright/test";
import { expect, test } from "../src/fixtures";

const { viewport, userAgent, hasTouch, isMobile, deviceScaleFactor } =
	devices["iPhone 14"];

test.describe("Draft review on the phone", () => {
	test.use({ viewport, userAgent, hasTouch, isMobile, deviceScaleFactor });

	test("pending draft settles into a diff, then undo and keep", async ({
		page,
		phoneDraftPage,
	}) => {
		await phoneDraftPage.goto("draft-pending");
		await expect(phoneDraftPage.status).toBeVisible();
		await expect(phoneDraftPage.undo.first()).toBeVisible({ timeout: 15_000 });
		await expect(phoneDraftPage.status).toBeHidden();
		await expect(page.getByRole("textbox")).toHaveCount(0);

		const before = await phoneDraftPage.undo.count();
		await phoneDraftPage.undo.first().click();
		await expect(phoneDraftPage.undo).toHaveCount(before - 1);

		await phoneDraftPage.keep.click();
		await expect(phoneDraftPage.kept).toBeVisible({ timeout: 15_000 });
		await expect(phoneDraftPage.undo).toHaveCount(0);
	});

	test("failed draft shows its error and a retry", async ({
		phoneDraftPage,
	}) => {
		await phoneDraftPage.goto("draft-failed");
		await expect(phoneDraftPage.error).toContainText("the model timed out");
		await expect(phoneDraftPage.retry).toBeVisible();
	});
});
