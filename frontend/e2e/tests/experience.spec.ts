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

	test("adds, edits and deletes an achievement variant", async ({
		experiencePage,
	}) => {
		const employer = `Hooli ${Date.now()}`;
		await experiencePage.addPosition(employer, "Engineer");
		await experiencePage.addAchievement(employer, "Shipped the thing");
		const card = experiencePage.positions.filter({ hasText: employer });

		await experiencePage.addVariant(employer, "Launched the widget early");
		await expect(card).toContainText("Launched the widget early");

		await experiencePage.editVariantText(
			employer,
			"Launched the widget early",
			"Shipped the widget ahead of schedule on a tight budget",
		);
		await expect(card).toContainText("Shipped the widget ahead of schedule…");

		await experiencePage.addVariant(employer, "Another take", "Short");
		await expect(card.getByTestId("variant")).toHaveCount(2);

		await experiencePage.deleteVariant(employer, "Short");
		await expect(card.getByTestId("variant")).toHaveCount(1);

		await experiencePage.deletePosition(employer);
		await expect(card).toHaveCount(0);
	});

	test("adds a Bank Skill under its category, then deletes it", async ({
		experiencePage,
	}) => {
		const name = `Zig ${Date.now()}`;

		await experiencePage.addBankSkill(name, "Languages");
		const skill = experiencePage.bankSkills.filter({ hasText: name });
		await expect(skill).toHaveCount(1);

		await experiencePage.deleteBankSkill(name);
		await expect(skill).toHaveCount(0);
	});
});
