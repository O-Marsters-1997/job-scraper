import { devices } from "@playwright/test";
import { seed } from "../../src/mocks/seed";
import { expect, test } from "../src/fixtures";

const withKeptDraft = seed.jobs.at(-1)!;

test.describe("CV sheet on a phone", () => {
	test.use({
		viewport: devices["iPhone 12"].viewport,
		hasTouch: true,
		isMobile: true,
	});

	test("lists visible base CVs and this Job's kept Draft", async ({
		jobDetailPage,
	}) => {
		await jobDetailPage.goto(withKeptDraft.ID);
		await jobDetailPage.openCvSheet();

		const sheet = jobDetailPage.dialog;
		await expect(sheet.getByText("Senior Backend Engineer CV")).toBeVisible();
		await expect(sheet.getByText("Platform Engineer CV")).toBeVisible();
		await expect(sheet.getByText("Cover Letter Template")).toBeHidden();
		await expect(
			sheet.getByRole("button", { name: /Kept Draft/ }),
		).toBeVisible();
	});

	test("downloads the PDF where files can't be shared", async ({
		page,
		jobDetailPage,
	}) => {
		await page.addInitScript(() => {
			Object.defineProperty(navigator, "canShare", { value: undefined });
		});
		await jobDetailPage.goto(withKeptDraft.ID);
		await jobDetailPage.openCvSheet();

		const download = page.waitForEvent("download");
		await jobDetailPage.dialog
			.getByRole("button", { name: /Platform Engineer CV/ })
			.dispatchEvent("click");
		expect((await download).suggestedFilename()).toBe(
			`cv-${withKeptDraft.CompanySlug}.pdf`,
		);
	});

	test("shares the PDF where files can be shared", async ({
		page,
		jobDetailPage,
	}) => {
		await page.addInitScript(() => {
			Object.defineProperty(navigator, "canShare", { value: () => true });
			Object.defineProperty(navigator, "share", {
				value: async (data: ShareData) => {
					(window as unknown as { shared: string }).shared =
						data.files?.[0]?.name ?? "";
				},
			});
		});
		await jobDetailPage.goto(withKeptDraft.ID);
		await jobDetailPage.openCvSheet();
		await jobDetailPage.dialog
			.getByRole("button", { name: /Kept Draft/ })
			.dispatchEvent("click");

		await expect
			.poll(() =>
				page.evaluate(() => (window as unknown as { shared?: string }).shared),
			)
			.toBe(`cv-${withKeptDraft.CompanySlug}.pdf`);
	});
});
