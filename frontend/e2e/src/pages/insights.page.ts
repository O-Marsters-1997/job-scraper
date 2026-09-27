import type { Locator, Page } from "@playwright/test";

export class InsightsPage {
	private readonly heading: Locator;
	readonly firstChartCanvas: Locator;

	constructor(private readonly page: Page) {
		this.heading = page.getByRole("heading", { name: "Insights" });
		this.firstChartCanvas = page.locator('canvas[role="img"]').first();
	}

	async gotoFromNav() {
		await this.page.getByRole("link", { name: "Insights" }).click();
		await this.heading.waitFor({ state: "visible" });
		await this.firstChartCanvas.waitFor({ state: "visible" });
	}

	snapshot(): Promise<string> {
		return this.firstChartCanvas.evaluate((el: HTMLCanvasElement) =>
			el.toDataURL(),
		);
	}
}
