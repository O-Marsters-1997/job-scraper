import type { Locator, Page } from "@playwright/test";

export class TweaksPanelComponent {
	readonly toggle: Locator;
	readonly panel: Locator;
	readonly customThemeButton: Locator;
	readonly hexInput: Locator;

	constructor(private readonly page: Page) {
		this.toggle = page.getByLabel("Open tweaks panel");
		this.panel = page.locator("div.fixed.bottom-4.right-4");
		this.customThemeButton = this.panel.getByRole("button", {
			name: "Custom",
		});
		this.hexInput = page.getByPlaceholder("#rrggbb");
	}

	async open() {
		await this.toggle.click();
	}

	async openCustomEditor() {
		await this.customThemeButton.click();
	}

	primaryColorTrigger(): Locator {
		return this.panel.getByLabel("Edit Primary", { exact: true });
	}

	async primaryColor(): Promise<string> {
		return this.page.evaluate(() =>
			getComputedStyle(document.documentElement)
				.getPropertyValue("--color-primary")
				.trim(),
		);
	}
}
