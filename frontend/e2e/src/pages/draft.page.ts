import type { Locator, Page } from "@playwright/test";

export class DraftPage {
	readonly saveStatus: Locator;
	readonly meter: Locator;
	readonly cards: Locator;
	readonly wand: Locator;
	readonly suggestion: Locator;

	constructor(readonly page: Page) {
		this.saveStatus = page.getByTestId("save-status");
		this.meter = page.getByTestId("page-meter");
		this.cards = page.getByTestId("margin-card");
		this.wand = page.getByRole("button", { name: "Edit with Haiku" });
		this.suggestion = page.getByTestId("suggestion-card");
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
