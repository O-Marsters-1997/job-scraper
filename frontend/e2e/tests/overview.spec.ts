import { getJobs } from "../../src/mocks/db";
import { expect, test } from "../src/fixtures";

test.describe("Overview", () => {
	test("Total jobs counts the full job list, not just the first page", async ({
		overviewPage,
	}) => {
		const seededCount = getJobs().length;
		expect(seededCount).toBeGreaterThan(100);

		await overviewPage.goto();

		await expect(overviewPage.totalJobsValue).toHaveText(
			String(seededCount),
		);
	});
});
