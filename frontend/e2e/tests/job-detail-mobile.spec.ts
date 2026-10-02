import { devices } from "@playwright/test";
import { seed } from "../../src/mocks/seed";
import { expect, test } from "../src/fixtures";

const untracked = seed.jobs[100]!;
const tracked = seed.applications[0]!;

test.describe("Job page on a phone", () => {
	test.use({
		viewport: devices["iPhone 12"].viewport,
		hasTouch: true,
		isMobile: true,
	});

	test("fits the viewport and keeps the action bar visible while scrolling", async ({
		page,
		jobDetailPage,
	}) => {
		await jobDetailPage.goto(untracked.ID);
		expect(await jobDetailPage.hasHorizontalScroll()).toBe(false);

		await page.mouse.wheel(0, 3000);
		await expect(jobDetailPage.actionBar).toBeInViewport();
		await expect(
			jobDetailPage.actionBar.getByRole("link", { name: "Open listing" }),
		).toHaveAttribute("href", untracked.URL);
	});

	test("I applied creates an Application with today's date", async ({
		page,
		jobDetailPage,
	}) => {
		await jobDetailPage.goto(untracked.ID);
		await expect(page.getByText("Not yet tracked")).toBeVisible();

		await jobDetailPage.tapApplied();

		await expect(page.getByText("Not yet tracked")).toBeHidden();
		await expect(
			page.getByText("Applied", { exact: true }).first(),
		).toBeVisible();
		const today = new Date().toLocaleDateString("en-GB", {
			day: "numeric",
			month: "short",
			year: "numeric",
		});
		await expect(page.getByText(today)).toBeVisible();
	});

	test("I applied on a tracked Job keeps notes and salary", async ({
		page,
		jobDetailPage,
	}) => {
		await jobDetailPage.goto(tracked.JobID);
		await page.getByRole("button", { name: "Edit", exact: true }).click();
		await jobDetailPage.dialog.getByLabel("Notes").fill("Spoke to recruiter");
		await jobDetailPage.dialog.getByLabel("Salary / comp").fill("£90,000");
		await jobDetailPage.dialog.getByRole("button", { name: "Update" }).click();
		await expect(jobDetailPage.dialog).toBeHidden();

		await jobDetailPage.tapApplied();

		await expect(
			page.getByText("Applied", { exact: true }).first(),
		).toBeVisible();
		await expect(page.getByText("Spoke to recruiter")).toBeVisible();
		await expect(page.getByText("£90,000")).toBeVisible();
	});

	test("I applied opens the track dialog when no status is named Applied", async ({
		page,
		jobDetailPage,
		statusesPage,
		mobileNav,
	}) => {
		await statusesPage.goto();
		await page.getByRole("button", { name: "Edit" }).nth(1).click();
		await page.getByLabel("Status name").fill("Submitted");
		await page.getByRole("button", { name: "Save" }).click();
		await expect(statusesPage.statusText("Submitted")).toBeVisible();

		await mobileNav.open();
		await mobileNav.link("Jobs").click();
		await page.getByRole("row").nth(1).getByRole("link").first().click();
		await expect(jobDetailPage.applied).toBeEnabled();
		await jobDetailPage.tapApplied();

		await expect(jobDetailPage.dialog).toBeVisible();
	});
});
