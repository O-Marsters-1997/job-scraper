import { devices } from "@playwright/test";
import { seed } from "../../src/mocks/seed";
import { expect, test } from "../src/fixtures";

const job = seed.jobs[0]!;

test.describe("Tailor from the phone", () => {
	test.use({
		viewport: devices["iPhone 12"].viewport,
		hasTouch: true,
		isMobile: true,
	});

	test("tailor, review, keep, download", async ({
		page,
		jobDetailPage,
		phoneDraftPage,
	}) => {
		await page.addInitScript(() => {
			Object.defineProperty(navigator, "canShare", { value: undefined });
		});
		await jobDetailPage.goto(job.ID);
		await jobDetailPage.openCvSheet();
		await jobDetailPage.dialog
			.getByRole("group", { name: "Platform Engineer CV" })
			.getByRole("button", { name: /Tailor/ })
			.dispatchEvent("click");

		await expect(page).toHaveURL(/\/tailoring\/drafts\/draft-/);
		await expect(phoneDraftPage.undo.first()).toBeVisible({ timeout: 15_000 });
		await phoneDraftPage.keep.click();
		await expect(phoneDraftPage.kept).toBeVisible({ timeout: 15_000 });

		const download = page.waitForEvent("download");
		await phoneDraftPage.download.click();
		expect((await download).suggestedFilename()).toBe(
			`cv-${job.CompanySlug}.pdf`,
		);
	});
});
