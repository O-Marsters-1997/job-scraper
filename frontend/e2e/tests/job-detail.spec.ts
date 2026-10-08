import { seed } from "../../src/mocks/seed";
import { expect, test } from "../src/fixtures";

const merged = seed.jobs.find((job) => (job.Listings?.length ?? 0) > 1)!;
const single = seed.jobs.find((job) => job.Listings?.length === 1)!;

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
