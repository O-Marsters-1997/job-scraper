import type { Locator, Page } from "@playwright/test";

export const SAVED_TIMEOUT_MS = 10_000;
const MAX_FILL_WORDS = 200;

export class DraftPage {
	readonly saveStatus: Locator;
	readonly meter: Locator;
	readonly cards: Locator;
	readonly wand: Locator;
	readonly suggestion: Locator;
	readonly suggestionDiff: Locator;
	readonly resolvedCount: Locator;
	readonly dots: Locator;

	constructor(readonly page: Page) {
		this.saveStatus = page.getByTestId("save-status");
		this.meter = page.getByTestId("page-meter");
		this.cards = page.getByTestId("margin-card");
		this.wand = page.getByRole("button", { name: "Edit with Haiku" });
		this.suggestion = page.getByTestId("suggestion-card");
		this.suggestionDiff = page.getByTestId("suggestion-diff");
		this.resolvedCount = page.getByTestId("resolved-count");
		this.dots = page.getByTestId("card-dot");
	}

	async goto(id = "draft-ready") {
		await this.page.goto(`/tailoring/drafts/${id}`);
		await this.page
			.getByRole("heading", { name: "Draft CV", level: 1 })
			.waitFor({ state: "visible" });
		await this.page.getByRole("textbox").first().waitFor({ state: "visible" });
	}

	line(label: string) {
		return this.page.getByRole("textbox", { name: label, exact: true });
	}

	card(key: string) {
		return this.page.locator(`[data-card-key="${key}"]`);
	}

	slot(slotId: string) {
		return this.page.locator(`[data-slot-id="${slotId}"]`);
	}

	lineCount(slotId: string) {
		return this.slot(slotId).getByTestId("line-count");
	}

	dot(key: string) {
		return this.page.locator(
			`[data-testid="card-dot"][data-card-key="${key}"]`,
		);
	}

	async fillToLines(label: string, slotId: string, lines: number) {
		const count = this.lineCount(slotId);
		for (let words = 20; words <= MAX_FILL_WORDS; words++) {
			await this.replace(label, "word ".repeat(words));
			await count.waitFor({ state: "attached" });
			const title = (await count.getAttribute("title")) ?? "";
			if (title.startsWith(`${lines} lines`)) return words;
		}
		throw new Error(`${label} never reached ${lines} lines`);
	}

	async type(label: string, text: string) {
		const line = this.line(label);
		await line.click();
		await this.page.keyboard.press("End");
		await this.page.keyboard.insertText(text);
	}

	async replace(label: string, text: string) {
		await this.line(label).fill(text);
	}

	async chooseWand(label: string, item: string) {
		await this.line(label).click();
		await this.wand.click();
		await this.page.getByRole("menuitem", { name: item }).click();
	}

	async lineCounts() {
		return this.page
			.getByTestId("line-count")
			.evaluateAll((els) => els.map((el) => el.getAttribute("title")));
	}
}

export class PhoneDraftPage {
	readonly status: Locator;
	readonly error: Locator;
	readonly retry: Locator;
	readonly undo: Locator;
	readonly keep: Locator;
	readonly kept: Locator;
	readonly download: Locator;

	constructor(readonly page: Page) {
		this.status = page.getByText("Tailoring…");
		this.error = page.getByRole("alert").filter({ hasText: "failed" });
		this.retry = page.getByRole("link", { name: "Try again" });
		this.undo = page.getByRole("button", { name: "Undo" });
		this.keep = page.getByRole("button", { name: "Keep", exact: true });
		this.kept = page.getByText("Kept", { exact: true });
		this.download = page.getByRole("button", { name: "Download" });
	}

	async goto(id: string) {
		await this.page.goto(`/tailoring/drafts/${id}`);
		await this.page
			.getByRole("heading", { name: "Draft CV", level: 1 })
			.waitFor({ state: "visible" });
	}
}
