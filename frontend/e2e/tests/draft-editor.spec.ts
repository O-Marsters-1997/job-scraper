import { expect, test } from "../src/fixtures";

const BULLET = "Senior Backend Engineer bullet 1";
const SECOND_BULLET = "Senior Backend Engineer bullet 2";

test.describe("Draft editor", () => {
	test.beforeEach(async ({ draftPage }) => {
		await draftPage.goto();
	});

	test("shows Saving after an edit, then Saved", async ({ draftPage }) => {
		await expect(draftPage.saveStatus).toHaveText("Saved");
		await draftPage.type(BULLET, " again");
		await expect(draftPage.saveStatus).toHaveText("Saving…");
		await expect(draftPage.saveStatus).toHaveText("Saved", { timeout: 10_000 });
	});

	test("keeps the caret while typing", async ({ page, draftPage }) => {
		await draftPage.line(BULLET).click();
		await page.keyboard.press("Home");
		await page.keyboard.type("AB");
		await expect(draftPage.line(BULLET)).toHaveText(/^ABCut p99/);
		await expect(draftPage.line(BULLET)).toBeFocused();
	});

	test("flips the meter when the page overflows", async ({ draftPage }) => {
		await expect(draftPage.meter).toContainText("One page");
		await draftPage.replace(BULLET, "word ".repeat(1200));
		await expect(draftPage.meter).toContainText("Spills onto page 2");
	});

	test("lets Google's page count override the estimate after a save", async ({
		draftPage,
	}) => {
		await draftPage.replace(BULLET, "word ".repeat(310));
		await expect(draftPage.meter).toContainText("One page");
		await expect(draftPage.meter).toContainText("Google renders 2 pages", {
			timeout: 10_000,
		});
		await draftPage.replace(BULLET, "short");
		await expect(draftPage.meter).toContainText("Google agrees", {
			timeout: 10_000,
		});
	});

	test("is read-only once the draft is kept", async ({ draftPage }) => {
		await draftPage.goto("draft-kept");
		await expect(draftPage.line(BULLET)).toHaveAttribute(
			"contenteditable",
			"false",
		);
		await expect(draftPage.saveStatus).toHaveCount(0);
	});

	test("shows an edit in the line's card", async ({ draftPage }) => {
		await draftPage.type(BULLET, " Terraform");
		await expect(draftPage.card("s1").getByTestId("card-diff")).toContainText(
			"Terraform",
		);
	});

	test("undoes edits and saves the restored text", async ({ draftPage }) => {
		const original = (await draftPage.line(BULLET).textContent()) ?? "";
		await draftPage.type(BULLET, " Terraform");
		await expect(draftPage.card("s1")).toContainText("Your edit");

		await draftPage
			.card("s1")
			.getByRole("button", { name: "Undo my edits" })
			.click();
		await expect(draftPage.line(BULLET)).toHaveText(original);
		await expect(draftPage.card("s1")).toHaveCount(0);
		await expect(draftPage.saveStatus).toHaveText("Saved", { timeout: 10_000 });
	});

	test("keeps cards apart and the active card level with its line", async ({
		page,
		draftPage,
	}) => {
		await draftPage.type(SECOND_BULLET, " again");
		await draftPage.card("s2").getByRole("button").first().click();
		await expect(
			draftPage.card("s2").getByRole("button", { name: "Resolve" }),
		).toBeVisible();

		const overlap = () =>
			draftPage.cards.evaluateAll((els) => {
				const boxes = els
					.map((el) => el.getBoundingClientRect())
					.sort((a, b) => a.top - b.top);
				return boxes.some(
					(r, i) => i > 0 && r.top < (boxes[i - 1]?.bottom ?? 0),
				);
			});
		await expect.poll(overlap).toBe(false);

		const gap = async () => {
			const lineTop = await page
				.locator('[data-slot-id="s2"]')
				.evaluate((el) => el.getBoundingClientRect().top);
			const cardTop = await draftPage
				.card("s2")
				.evaluate((el) => el.getBoundingClientRect().top);
			return Math.abs(cardTop - lineTop);
		};
		await expect.poll(gap).toBeLessThan(2);
	});

	test("hints at a cut only for a sparse last line", async ({
		page,
		draftPage,
	}) => {
		const counts = page.locator(
			'[data-slot-id="s1"] [data-testid="line-count"]',
		);
		const words = async (n: number) => {
			await draftPage.replace(BULLET, "word ".repeat(n));
			await expect(counts).toHaveAttribute("title", /lines/);
			return (await counts.getAttribute("title")) ?? "";
		};
		let n = 20;
		while (!(await words(n)).startsWith("3 lines")) n++;

		await draftPage.card("s1").getByRole("button").first().click();
		await expect(draftPage.card("s1").getByTestId("card-fit")).toBeVisible();

		await words(n + 8);
		await expect(draftPage.card("s1").getByTestId("card-fit")).toHaveCount(0);
	});

	test("counts resolved cards", async ({ page, draftPage }) => {
		await expect(page.getByTestId("resolved-count")).toContainText("0/");
		await draftPage.type(BULLET, " Terraform");
		await draftPage.card("s1").getByRole("button").first().click();
		await draftPage.card("s1").getByRole("button", { name: "Resolve" }).click();
		await expect(page.getByTestId("resolved-count")).toContainText("1/");
	});

	test("tightens a line through the wand and saves it on Accept", async ({
		draftPage,
	}) => {
		const before = (await draftPage.line(BULLET).textContent()) ?? "";
		await draftPage.chooseWand(BULLET, "Tighten");
		await expect(draftPage.page.getByTestId("suggestion-diff")).toBeVisible();
		await expect(
			draftPage.suggestion.getByRole("button", { name: "Accept" }),
		).toBeVisible();
		await draftPage.suggestion.getByRole("button", { name: "Accept" }).click();

		await expect(draftPage.suggestion).toHaveCount(0);
		await expect(draftPage.line(BULLET)).not.toHaveText(before);
		expect(before).toContain(
			((await draftPage.line(BULLET).textContent()) ?? "").slice(0, 20),
		);
		await expect(draftPage.saveStatus).toHaveText("Saved", { timeout: 10_000 });
	});

	test("an emptied line stays unsaved while other edits save", async ({
		draftPage,
	}) => {
		await draftPage.replace(BULLET, "");
		await draftPage.type(SECOND_BULLET, " again");
		await expect(draftPage.saveStatus).toHaveText("Saved", { timeout: 10_000 });
		await expect(draftPage.card("s1")).toContainText("Your edit");
	});

	test("restores the line on Reject", async ({ draftPage }) => {
		const before = (await draftPage.line(BULLET).textContent()) ?? "";
		await draftPage.chooseWand(BULLET, "Stronger verb");
		await draftPage.suggestion.getByRole("button", { name: "Reject" }).click();
		await expect(draftPage.suggestion).toHaveCount(0);
		await expect(draftPage.line(BULLET)).toHaveText(before);
	});

	test("rejects on Escape and asks with the keyboard", async ({
		page,
		draftPage,
	}) => {
		await draftPage.chooseWand(BULLET, "Tighten");
		await expect(draftPage.suggestion).toBeVisible();
		await page.keyboard.press("Escape");
		await expect(draftPage.suggestion).toHaveCount(0);

		const before = (await draftPage.line(BULLET).textContent()) ?? "";
		await draftPage.line(BULLET).click();
		await page.keyboard.press("ControlOrMeta+k");
		await page
			.getByRole("textbox", { name: "Ask Haiku to edit this line" })
			.fill("make it concise");
		await page.keyboard.press("Enter");
		await expect(draftPage.suggestion).toContainText('"make it concise"');
		await expect(
			draftPage.suggestion.getByRole("button", { name: "Accept" }),
		).toBeVisible();
		await page.keyboard.press("ControlOrMeta+Enter");
		await expect(draftPage.suggestion).toHaveCount(0);
		await expect(draftPage.line(BULLET)).not.toHaveText(before);
		await expect(draftPage.saveStatus).toHaveText("Saved", { timeout: 10_000 });
	});

	test("offers Fit only for a wrapped line, and Accept saves a line", async ({
		page,
		draftPage,
	}) => {
		await draftPage.line(BULLET).click();
		await draftPage.replace(BULLET, "short");
		await draftPage.wand.click();
		await expect(
			page.getByRole("menuitem", { name: "Fit to fewer lines" }),
		).toHaveCount(0);
		await page.keyboard.press("Escape");

		const counts = page.locator(
			'[data-slot-id="s1"] [data-testid="line-count"]',
		);
		await draftPage.replace(BULLET, "word ".repeat(40));
		await expect(counts).toHaveAttribute("title", /^[2-9] lines/);
		const lines = Number(((await counts.getAttribute("title")) ?? "")[0]);

		await draftPage.chooseWand(BULLET, "Fit to fewer lines");
		await expect(draftPage.suggestion).toContainText("Saves");
		await draftPage.suggestion.getByRole("button", { name: "Accept" }).click();
		await expect(counts).toHaveAttribute(
			"title",
			new RegExp(`^${lines - 1} lines?`),
		);
	});

	test("the margin card's Fit button starts a suggestion", async ({
		page,
		draftPage,
	}) => {
		const counts = page.locator(
			'[data-slot-id="s1"] [data-testid="line-count"]',
		);
		let n = 20;
		for (;;) {
			await draftPage.replace(BULLET, "word ".repeat(n));
			await expect(counts).toHaveAttribute("title", /lines/);
			if (((await counts.getAttribute("title")) ?? "").startsWith("3 lines"))
				break;
			n++;
		}
		await draftPage.card("s1").getByRole("button").first().click();
		await draftPage.card("s1").getByRole("button", { name: "Fit" }).click();
		await expect(draftPage.suggestion).toBeVisible();
	});

	test("announces a ready suggestion", async ({ draftPage }) => {
		await draftPage.chooseWand(BULLET, "Tighten");
		await expect(draftPage.page.getByText("Suggestion ready")).toBeAttached();
	});
});

for (const width of [375, 1024]) {
	test.describe(`Draft editor at ${width}px`, () => {
		test.use({ viewport: { width, height: 800 } });

		test("has no horizontal scroll and keeps the desktop line counts", async ({
			page,
			draftPage,
		}) => {
			await page.setViewportSize({ width: 1280, height: 800 });
			await draftPage.goto();
			await expect(
				draftPage.page.getByTestId("line-count").first(),
			).toBeVisible();
			const desktopCounts = await draftPage.lineCounts();
			const desktopMeter = await draftPage.meter.textContent();

			await page.setViewportSize({ width, height: 800 });
			await expect(page.getByTestId("card-dot").first()).toBeVisible();
			await expect(draftPage.cards).toHaveCount(0);
			expect(await draftPage.lineCounts()).toEqual(desktopCounts);
			await expect(draftPage.meter).toHaveText(desktopMeter ?? "");
			const overflow = await page.evaluate(
				() => document.documentElement.scrollWidth - window.innerWidth,
			);
			expect(overflow).toBeLessThanOrEqual(0);
		});

		test("opens the active card inline and reaches the wand", async ({
			page,
			draftPage,
		}) => {
			await draftPage.goto();
			await expect(page.getByTestId("card-dot").first()).toBeVisible();
			await draftPage.type(BULLET, " again");

			const card = draftPage.card("s1");
			await expect(card.getByRole("button", { name: "Resolve" })).toBeVisible();
			const gapBelowLine = async () => {
				const c = await card.boundingBox();
				const l = await page.locator('[data-slot-id="s1"]').boundingBox();
				return (c?.y ?? 0) - ((l?.y ?? 0) + (l?.height ?? 0));
			};
			await expect.poll(gapBelowLine).toBeGreaterThanOrEqual(0);
			const box = await card.boundingBox();
			expect(box?.x ?? -1).toBeGreaterThanOrEqual(0);
			expect((box?.x ?? 0) + (box?.width ?? width + 1)).toBeLessThanOrEqual(
				width,
			);

			await draftPage.wand.click();
			await page.getByRole("menuitem", { name: "Tighten" }).click();
			await draftPage.suggestion
				.getByRole("button", { name: "Accept" })
				.click();
			await expect(draftPage.suggestion).toHaveCount(0);
		});

		test("activates a line from its gutter dot", async ({
			page,
			draftPage,
		}) => {
			await draftPage.goto();
			await page
				.locator('[data-testid="card-dot"][data-card-key="s2"]')
				.click();
			await expect(draftPage.cards).toHaveCount(1);
			await expect(
				draftPage.cards.first().getByRole("button", { name: "Resolve" }),
			).toBeVisible();
		});
	});
}
