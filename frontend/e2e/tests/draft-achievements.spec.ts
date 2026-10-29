import { expect, test } from "../src/fixtures";
import { SAVED_TIMEOUT_MS } from "../src/pages/draft.page";

const SECOND =
	"Led the migration of twelve services to Kubernetes, reducing deploy time and the pager load for a team of eight across two time zones.";
const BANK_SECOND = "Led the migration of 12 services from EC2 to Kubernetes";
const BANK_FIRST =
	"Cut p99 API latency by 40% by moving hot paths to a read-through cache";
const BANK_THIRD =
	"Mentored three engineers through their first on-call rotation";
const PICK_TOGGLE = { name: /^(Add|Remove)$/ };

test.describe("Achievements drawer", () => {
	test.beforeEach(async ({ draftPage }) => {
		await draftPage.goto();
		await draftPage.openRole();
	});

	test("lists the role's bank by fit with its picks numbered", async ({
		draftPage,
	}) => {
		const rows = draftPage.drawer.getByRole("option");
		await expect(rows).toHaveCount(3);
		await expect(rows.nth(0)).toContainText(BANK_FIRST);
		await expect(rows.nth(0)).toContainText("91%");
		await expect(rows.nth(0).getByRole("button", PICK_TOGGLE)).toHaveText("1");
		await expect(rows.nth(1).getByRole("button", PICK_TOGGLE)).toHaveText("2");
		await expect(rows.nth(2)).toContainText("Unclear");
		await expect(rows.nth(2).getByRole("button", PICK_TOGGLE)).toHaveText("");
	});

	test("a pick change rewrites the role's bullets and saves", async ({
		page,
		draftPage,
	}) => {
		await draftPage.row(BANK_FIRST).getByRole("button", PICK_TOGGLE).click();
		await expect(page.getByRole("textbox", { name: /bullet/ })).toHaveCount(2);
		await expect(draftPage.line("Senior Backend Engineer bullet 1")).toHaveText(
			SECOND,
		);
		await expect(draftPage.saveStatus).toHaveText("Saved", {
			timeout: SAVED_TIMEOUT_MS,
		});
	});

	test("clicking an unpicked row adds it last with its bank text and no edit card", async ({
		page,
		draftPage,
	}) => {
		await draftPage.row(BANK_THIRD).click();
		await expect(
			draftPage.row(BANK_THIRD).getByRole("button", PICK_TOGGLE),
		).toHaveText("3");
		await expect(draftPage.line("Senior Backend Engineer bullet 3")).toHaveText(
			BANK_THIRD,
			{ timeout: SAVED_TIMEOUT_MS },
		);
		await draftPage.drawer.getByRole("button", { name: "Close" }).click();
		await expect(page.getByText("Your edit")).toHaveCount(0);

		await draftPage.type("Senior Backend Engineer bullet 3", " again");
		await expect(page.getByText("Your edit")).toHaveCount(1);
	});

	test("re-picking a generated Achievement restores the generated text", async ({
		page,
		draftPage,
	}) => {
		await draftPage.row(BANK_SECOND).getByRole("button", PICK_TOGGLE).click();
		await expect(
			page.getByRole("textbox").filter({ hasText: SECOND }),
		).toHaveCount(0);
		await draftPage.row(BANK_SECOND).click();
		await expect(
			page.getByRole("textbox").filter({ hasText: SECOND }),
		).toHaveCount(1);
	});

	test("keys move the selection and the picks, and Esc steps back", async ({
		draftPage,
	}) => {
		const list = draftPage.drawer.getByRole("listbox");
		await expect(list).toBeFocused();
		await list.press("ArrowDown");
		await expect(draftPage.drawer.getByRole("option").nth(0)).toHaveAttribute(
			"aria-selected",
			"true",
		);
		await expect(
			draftPage.row(BANK_FIRST).getByRole("button", PICK_TOGGLE),
		).toHaveText("1");

		await list.press("2");
		await expect(
			draftPage.row(BANK_FIRST).getByRole("button", PICK_TOGGLE),
		).toHaveText("2");
		await expect(draftPage.line("Senior Backend Engineer bullet 1")).toHaveText(
			SECOND,
		);

		await list.press("ArrowDown");
		await list.press("ArrowDown");
		await expect(draftPage.row(BANK_THIRD)).toHaveAttribute(
			"aria-selected",
			"true",
		);
		await expect(
			draftPage.row(BANK_THIRD).getByRole("button", PICK_TOGGLE),
		).toHaveText("");

		await list.press("Space");
		await expect(
			draftPage.row(BANK_THIRD).getByRole("button", PICK_TOGGLE),
		).toHaveText("3");

		await list.press("Escape");
		await expect(
			draftPage.drawer.locator('[aria-selected="true"]'),
		).toHaveCount(0);
		await expect(draftPage.drawer).toBeVisible();
		await list.press("Escape");
		await expect(draftPage.drawer).toBeHidden();
	});

	test("the line budget follows the page", async ({ draftPage }) => {
		const before = await draftPage.usedLines();
		await draftPage.row(BANK_SECOND).getByRole("button", PICK_TOGGLE).click();
		await expect.poll(() => draftPage.usedLines()).toBeLessThan(before);
	});

	test("closes on a click outside", async ({ page, draftPage }) => {
		await page.getByRole("heading", { name: "Draft CV" }).click();
		await expect(draftPage.drawer).toBeHidden();
	});
});
