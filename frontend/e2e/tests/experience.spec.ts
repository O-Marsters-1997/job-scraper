import { expect, test } from "../src/fixtures";

test.describe("Experience", () => {
	test.beforeEach(async ({ experiencePage }) => {
		await experiencePage.goto();
	});

	test("adds a position with an achievement, then deletes it", async ({
		experiencePage,
	}) => {
		const employer = `Initech ${Date.now()}`;

		await experiencePage.addPosition(employer, "Platform Engineer");
		await experiencePage.addAchievement(employer, "Shipped the thing");

		const card = experiencePage.positions.filter({ hasText: employer });
		await expect(card).toContainText("Shipped the thing");
		await expect(experiencePage.positions.first()).toContainText(employer);

		await experiencePage.deletePosition(employer);
		await expect(card).toHaveCount(0);
	});
});
