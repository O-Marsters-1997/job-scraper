import { seed } from "../../src/mocks/seed";
import { expect, test } from "../src/fixtures";

const merged = seed.jobs.find((job) => (job.Listings?.length ?? 0) > 1)!;
const single = seed.jobs.find((job) => job.Listings?.length === 1)!;

test.describe("Job page tasks", () => {
	test("adds an Apply task, keeping the dialog open on an invalid link", async ({
		jobDetailPage,
		page,
	}) => {
		await jobDetailPage.goto(single.ID);
		await page.getByRole("button", { name: "Add task" }).click();
		const dialog = jobDetailPage.dialog;
		await expect(dialog.getByLabel("Type")).toHaveValue("task-type-apply");

		await dialog.getByLabel("Link").fill("ftp://nope");
		await dialog.getByRole("button", { name: "Save" }).click();
		await expect(dialog.getByRole("alert")).toContainText("http or https");

		await dialog.getByLabel("Link").fill("https://example.com/apply");
		await dialog.getByLabel("Notes").fill("send CV");
		await dialog.getByRole("button", { name: "Save" }).click();
		await expect(dialog).toBeHidden();
		await expect(page.getByText("send CV")).toBeVisible();
	});
});

test.describe("Job page listings", () => {
	test("links every other listing of a merged Job", async ({
		jobDetailPage,
	}) => {
		const secondary = merged.Listings![1]!;
		await jobDetailPage.goto(merged.ID);

		await expect(jobDetailPage.alsoOn).toBeVisible();
		const link = jobDetailPage.alsoOn.getByRole("link");
		await expect(link).toHaveCount(1);
		await expect(link).toHaveAttribute("href", secondary.url);
		await expect(link).toHaveAttribute("target", "_blank");
	});

	test("shows no Also on line for a single listing", async ({
		jobDetailPage,
	}) => {
		await jobDetailPage.goto(single.ID);
		await expect(jobDetailPage.alsoOn).toBeHidden();
	});
});
